package vtbackground

import (
	"context"
	"flag"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	vtinsert "github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	vtselect "github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
)

var (
	enableServiceGraph     = flag.Bool("servicegraph.enable", false, "Whether to enable background task for generating service graph.")
	serviceGraphInterval   = flag.Duration("servicegraph.taskInterval", time.Minute, "The background task interval for generating service graph data. It requires setting `-servicegraph.enable=true`.")
	serviceGraphLookbehind = flag.Duration("servicegraph.Lookbehind", time.Minute, "The lookbehind window for each time service graph background task run. It requires setting `-servicegraph.enable=true`.")
)

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
	r := &http.Request{}

	endTime := time.Now().Truncate(*serviceGraphInterval)
	startTime := endTime.Add(-*serviceGraphLookbehind)

	rows, err := vtselect.GetServiceGraphTimeRange(ctx, r, startTime, endTime)
	if err != nil {
		logger.Errorf("cannot get service graph for time range [%d, %d]: %s", startTime.Unix(), endTime.Unix(), err)
		return
	}
	if len(rows) == 0 {
		return
	}

	err = vtinsert.PersistServiceGraph(ctx, r, rows, endTime)
	if err != nil {
		logger.Errorf("cannot presist service graph for time %d: %s", endTime.Unix(), err)
	}
	return
}
