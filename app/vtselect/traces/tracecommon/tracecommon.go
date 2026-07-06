package tracecommon

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

var (
	TraceMaxDurationWindow = flag.Duration("search.traceMaxDurationWindow", 1*time.Minute, "The window of searching for the rest trace spans after finding one span."+
		"It allows extending the search start time and end time by -search.traceMaxDurationWindow to make sure all spans are included."+
		"It affects both Jaeger's /api/traces and /api/traces/<trace_id> APIs.")

	TraceSearchStep = flag.Duration("search.traceSearchStep", 24*time.Hour, "Splits the [0, now] time range into many small time ranges by -search.traceSearchStep "+
		"when searching for spans by trace_id. Once it finds spans in a time range, it performs an additional search according to -search.traceMaxDurationWindow and then stops. "+
		"It affects Jaeger's /api/traces/<trace_id> API.")
	TraceTagsLookbehind = flag.Duration("search.tagsLookbehind", 3*24*time.Hour, "The default time range of searching for tags and attributes."+
		"It affects Jaeger's /api/services and /api/services/*/operations APIs, and various Tempo tag-related APIs.")
	TraceMaxTraces = flag.Int("search.maxTraces", 1000, "The maximum number of traces that can be returned in a single search request. "+
		"Users may request with different limit value via query argument which shouldn't exceed this limit. This limit applies to Jaeger’s /api/traces API and Tempo's /api/search API.")
	TraceMaxTags = flag.Int("search.maxTags", 1000, "The maximum number of tags (including service name, span name) that can be returned in a single search request. "+
		"This limit applies to Jaeger’s /api/services, /api/services/*/operations APIs, and various Tempo tag-related APIs.")

	LatencyOffset = flag.Duration("search.latencyOffset", 30*time.Second, "The time when a trace become visible in query results after the collection. see -insert.traceMaxDuration as well. (default 30s)")

	// deprecated flags. preserve here for backward compatibility. should be removed in future version.
	_ = flag.Uint64("search.traceMaxServiceNameList", 1000, "Deprecated, see -search.maxTags.")
	_ = flag.Uint64("search.traceMaxSpanNameList", 1000, "Deprecated, see -search.maxTags.")
	_ = flag.Duration("search.traceServiceAndSpanNameLookbehind", 3*24*time.Hour, "Deprecated, see -search.tagsLookbehind.")
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

	hiddenFieldsFilters, err := getStringSliceFromRequest(r, "hidden_fields_filters")
	if err != nil {
		return nil, err
	}

	cp := &CommonParams{
		TenantIDs:           tenantIDs,
		HiddenFieldsFilters: hiddenFieldsFilters,
	}

	return cp, nil
}

func getStringSliceFromRequest(r *http.Request, argName string) ([]string, error) {
	s := r.FormValue(argName)
	if s == "" {
		return nil, nil
	}

	if strings.HasPrefix(s, "[") {
		// Parse as a JSON array of strings.
		var a []string
		if err := json.Unmarshal([]byte(s), &a); err != nil {
			return nil, fmt.Errorf("cannot unmarshal JSON array from %s=%q: %w", argName, s, err)
		}
		return a, nil
	}

	// Parse as a comma-separated list of strings
	a := strings.Split(s, ",")
	return a, nil
}

// Row represent the query result of a trace span.
type Row struct {
	Timestamp int64
	Fields    []logstorage.Field
}
