package query

import (
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
)

func TestCheckTraceIDList(t *testing.T) {
	f := func(traceID string, valid bool) {
		t.Helper()

		result := checkTraceIDList([]string{traceID})
		if valid != (len(result) == 1) {
			t.Fatalf("check trace id unexpected result, trace_id: %s, valid: %t", traceID, len(result) == 1)
		}
	}
	f("12345678", true)
	f("abcd1234567", true)
	f("asdf-asdf-1234-asdf", true)
	f("abcd1234:4321bcda:4321bacd", true)
	f("abcd.abcd.1234.4321", true)
	f("abcd bcad", false)
	f("abcd\"", false)
}

func TestInvertedTagFilterQueryParses(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	for _, qStr := range []string{
		`* AND !("span_attr:foo":re("pat")) | last 1 by (_time) partition by (trace_id) | fields _time, trace_id | sort by (_time) desc`,
		`* AND !("span_attr:foo":="bar") | last 1 by (_time) partition by (trace_id) | fields _time, trace_id | sort by (_time) desc`,
	} {
		if _, err := logstorage.ParseQueryAtTimestamp(qStr, ts); err != nil {
			t.Fatalf("parse %q: %v", qStr, err)
		}
	}
}
