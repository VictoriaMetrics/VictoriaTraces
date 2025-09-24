package vtbackground

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var (
	enableServiceGraph     = flag.Bool("servicegraph.enable", false, "Whether to enable background task for generating service graph.")
	serviceGraphInterval   = flag.Duration("servicegraph.taskInterval", time.Minute, "The background task interval for generating service graph data. It requires setting `-servicegraph.enable=true`.")
	serviceGraphLookbehind = flag.Duration("servicegraph.Lookbehind", time.Minute, "The lookbehind window for each time service graph background task run. It requires setting `-servicegraph.enable=true`.")
)

// Row represent the query result of a trace span.
type Row struct {
	Fields []logstorage.Field
}

type dependencyLink struct {
	parent    string
	child     string
	callCount uint64
}

func InitServiceGraph() {
	if !*enableServiceGraph {
		return
	}
	ticker := time.NewTicker(*serviceGraphInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				GetServiceGraphLastMin(context.TODO())
			}
		}
	}()
}

func GetServiceGraphLastMin(ctx context.Context) {
	qStrChildSpans := fmt.Sprintf(
		`(NOT %s:"") AND (%s:~"%d|%d")  | fields %s, %s | rename %s as %s, %s as child`,
		otelpb.ParentSpanIDField,
		otelpb.KindField,
		otelpb.SpanKind(2),
		otelpb.SpanKind(5),
		otelpb.ParentSpanIDField,
		otelpb.ResourceAttrServiceName,
		otelpb.ParentSpanIDField,
		otelpb.SpanIDField,
		otelpb.ResourceAttrServiceName,
	)
	qStrParentSpans := fmt.Sprintf(
		`(NOT %s:"") AND (%s:~"%d|%d") | fields %s, %s | rename %s as parent`,
		otelpb.SpanIDField,
		otelpb.KindField,
		otelpb.SpanKind(3),
		otelpb.SpanKind(4),
		otelpb.SpanIDField,
		otelpb.ResourceAttrServiceName,
		otelpb.ResourceAttrServiceName,
	)
	qStr := fmt.Sprintf(
		`%s | join by (%s) (%s) inner | NOT parent:eq_field(child) | stats by (parent, child) count() callCount`,
		qStrChildSpans,
		otelpb.SpanIDField,
		qStrParentSpans,
	)

	endTime := time.Now().Truncate(-*serviceGraphInterval)
	startTime := endTime.Add(-*serviceGraphLookbehind)

	q, err := logstorage.ParseQueryAtTimestamp(qStr, endTime.UnixNano())
	if err != nil {
		logger.Errorf("cannot parse query [%s]: %s", qStr, err)
		return
	}
	q.AddTimeFilter(startTime.UnixNano(), endTime.UnixNano())
	q.AddPipeOffsetLimit(0, 1000)

	qs := &logstorage.QueryStats{}

	qctx := logstorage.NewQueryContext(ctx, qs, []logstorage.TenantID{{}}, q)

	var rowsLock sync.Mutex
	var rows []*Row
	//var missingTimeColumn atomic.Bool
	writeBlock := func(_ uint, db *logstorage.DataBlock) {
		columns := db.Columns
		if len(columns) == 0 {
			return
		}
		clonedColumnNames := make([]string, len(columns))
		valuesCount := 0
		for i, c := range columns {
			clonedColumnNames[i] = strings.Clone(c.Name)
			if len(c.Values) > valuesCount {
				valuesCount = len(c.Values)
			}
		}
		if valuesCount == 0 {
			return
		}
		for i := 0; i < valuesCount; i++ {
			fields := make([]logstorage.Field, 0, len(columns))
			for j := range columns {
				fields = append(
					fields,
					logstorage.Field{
						Name:  clonedColumnNames[j],
						Value: strings.Clone(columns[j].Values[i]),
					},
				)
			}
			rowsLock.Lock()
			rows = append(rows, &Row{
				Fields: fields,
			})
			rowsLock.Unlock()
		}
	}

	if err = vtstorage.RunQuery(qctx, writeBlock); err != nil {
		logger.Errorf("cannot execute query [%s]: %s", qStr, err)
		return
	}

	if len(rows) == 0 {
		return
	}

	r := &http.Request{}
	cp, _ := insertutil.GetCommonParams(r)
	lmp := cp.NewLogMessageProcessor("background_task", false)

	for _, row := range rows {
		f := append(row.Fields, logstorage.Field{
			Name:  "_msg",
			Value: "-",
		})
		lmp.AddRow(endTime.UnixNano(), f, []logstorage.Field{{"service_graph_stream", "-"}})
	}
	lmp.MustClose()

	return
}
