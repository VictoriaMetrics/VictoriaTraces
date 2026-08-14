// Package extrafilters parses the extra_filters and extra_stream_filters query args.
//
// The args are shared by the native LogsQL endpoints and the Jaeger and Tempo APIs,
// so the parsing lives here rather than in one of them.
package extrafilters

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/valyala/fastjson"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
)

// ParseExtraFilters parses the extra_filters query arg.
func ParseExtraFilters(s string) (*logstorage.Filter, error) {
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, `{"`) {
		return logstorage.ParseFilter(s)
	}

	// Extra filters in the form {"field":"value",...}.
	kvs, err := parseExtraFiltersJSON(s)
	if err != nil {
		return nil, err
	}

	filters := make([]string, len(kvs))
	for i, kv := range kvs {
		if len(kv.values) == 1 {
			filters[i] = fmt.Sprintf("%q:=%q", kv.key, kv.values[0])
		} else {
			orValues := make([]string, len(kv.values))
			for j, v := range kv.values {
				orValues[j] = fmt.Sprintf("%q", v)
			}
			filters[i] = fmt.Sprintf("%q:in(%s)", kv.key, strings.Join(orValues, ","))
		}
	}
	s = strings.Join(filters, " ")
	return logstorage.ParseFilter(s)
}

// ParseExtraStreamFilters parses the extra_stream_filters query arg.
func ParseExtraStreamFilters(s string) (*logstorage.Filter, error) {
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, `{"`) {
		return logstorage.ParseFilter(s)
	}

	// Extra stream filters in the form {"field":"value",...}.
	kvs, err := parseExtraFiltersJSON(s)
	if err != nil {
		return nil, err
	}

	filters := make([]string, len(kvs))
	for i, kv := range kvs {
		if len(kv.values) == 1 {
			filters[i] = fmt.Sprintf("%q=%q", kv.key, kv.values[0])
		} else {
			orValues := make([]string, len(kv.values))
			for j, v := range kv.values {
				orValues[j] = regexp.QuoteMeta(v)
			}
			filters[i] = fmt.Sprintf("%q=~%q", kv.key, strings.Join(orValues, "|"))
		}
	}
	s = "{" + strings.Join(filters, ",") + "}"
	return logstorage.ParseFilter(s)
}

type extraFilter struct {
	key    string
	values []string
}

func parseExtraFiltersJSON(s string) ([]extraFilter, error) {
	v, err := fastjson.Parse(s)
	if err != nil {
		return nil, err
	}
	o := v.GetObject()

	var errOuter error
	var filters []extraFilter
	o.Visit(func(k []byte, v *fastjson.Value) {
		if errOuter != nil {
			return
		}
		switch v.Type() {
		case fastjson.TypeString:
			filters = append(filters, extraFilter{
				key:    string(k),
				values: []string{string(v.GetStringBytes())},
			})
		case fastjson.TypeArray:
			a := v.GetArray()
			if len(a) == 0 {
				return
			}
			orValues := make([]string, len(a))
			for i, av := range a {
				ov, err := av.StringBytes()
				if err != nil {
					errOuter = fmt.Errorf("cannot obtain string item at the array for key %q; item: %s", k, av)
					return
				}
				orValues[i] = string(ov)
			}
			filters = append(filters, extraFilter{
				key:    string(k),
				values: orValues,
			})
		default:
			errOuter = fmt.Errorf("unexpected type of value for key %q: %s; value: %s", k, v.Type(), v)
		}
	})
	if errOuter != nil {
		return nil, errOuter
	}
	return filters, nil
}
