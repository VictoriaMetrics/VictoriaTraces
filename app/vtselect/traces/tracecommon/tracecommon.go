package tracecommon

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

var (
	TraceMaxDurationWindow = flag.Duration("search.traceMaxDurationWindow", 1*time.Minute, "The window of searching for the rest trace spans after finding one span."+
		"It allows extending the search start time and end time by -search.traceMaxDurationWindow to make sure all spans are included."+
		"It affects both Jaeger's /api/traces and /api/traces/<trace_id> APIs.")
	TraceServiceAndSpanNameLookbehind = flag.Duration("search.traceServiceAndSpanNameLookbehind", 3*24*time.Hour, "The time range of searching for service name and span name. "+
		"It affects Jaeger's /api/services and /api/services/*/operations APIs.")
	TraceSearchStep = flag.Duration("search.traceSearchStep", 24*time.Hour, "Splits the [0, now] time range into many small time ranges by -search.traceSearchStep "+
		"when searching for spans by trace_id. Once it finds spans in a time range, it performs an additional search according to -search.traceMaxDurationWindow and then stops. "+
		"It affects Jaeger's /api/traces/<trace_id> API.")
	TraceMaxServiceNameList = flag.Uint64("search.traceMaxServiceNameList", 1000, "The maximum number of service name can return in a get service name request. "+
		"This limit affects Jaeger's /api/services API.")
	TraceMaxSpanNameList = flag.Uint64("search.traceMaxSpanNameList", 1000, "The maximum number of span name can return in a get span name request. "+
		"This limit affects Jaeger's /api/services/*/operations API.")

	LatencyOffset = flag.Duration("search.latencyOffset", 30*time.Second, "The time when a trace become visible in query results after the collection. see -insert.traceMaxDuration as well. (default 30s)")
)

var (
	TraceIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-.:]*$`)
)

// CommonParams common query params that shared by all requests.
type CommonParams struct {
	TenantIDs []logstorage.TenantID
	Query     *logstorage.Query

	// Whether to disable compression of the response sent to the vtselect.
	DisableCompression bool

	// Whether to allow partial response when some of vtstorage nodes are unavailable.
	AllowPartialResponse bool

	// Optional list of log fields or log field prefixes ending with *, which must be hidden during query execution.
	HiddenFieldsFilters []string

	// qs contains execution statistics for the Query.
	qs logstorage.QueryStats
}

func (cp *CommonParams) NewQueryContext(ctx context.Context) *logstorage.QueryContext {
	return logstorage.NewQueryContext(ctx, &cp.qs, cp.TenantIDs, cp.Query, cp.AllowPartialResponse, cp.HiddenFieldsFilters)
}

func (cp *CommonParams) UpdatePerQueryStatsMetrics() {
	vtstorage.UpdatePerQueryStatsMetrics(&cp.qs)
}

// GetCommonParams get common params from request for all traces query APIs.
func GetCommonParams(r *http.Request) (*CommonParams, error) {
	tenantID, err := logstorage.GetTenantIDFromRequest(r)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain tenantID: %w", err)
	}
	tenantIDs := []logstorage.TenantID{tenantID}
	cp := &CommonParams{
		TenantIDs: tenantIDs,
	}
	return cp, nil
}

// Row represent the query result of a trace span.
type Row struct {
	Timestamp int64
	Fields    []logstorage.Field
}
