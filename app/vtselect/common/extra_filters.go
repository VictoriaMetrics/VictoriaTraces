package common

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/valyala/fastjson"
)

// ParseExtraFilters parses extra_filters from either LogsQL or JSON format.
func ParseExtraFilters(s string) (*logstorage.Filter, error) {
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, `{"`) {
		return logstorage.ParseFilter(s)
	}

	// Extra filters in the form {"field":"value",...}.
	filters, err := parseExtraFiltersJSON(s)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(filters))
	for i, f := range filters {
		if len(f.values) == 1 {
			result[i] = fmt.Sprintf("%q:=%q", f.key, f.values[0])
		} else {
			orValues := make([]string, len(f.values))
			for j, v := range f.values {
				orValues[j] = fmt.Sprintf("%q", v)
			}
			result[i] = fmt.Sprintf("%q:in(%s)", f.key, strings.Join(orValues, ","))
		}
	}
	return logstorage.ParseFilter(strings.Join(result, " "))
}

// ParseExtraStreamFilters parses extra_stream_filters from either LogsQL or JSON format.
func ParseExtraStreamFilters(s string) (*logstorage.Filter, error) {
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, `{"`) {
		return logstorage.ParseFilter(s)
	}

	// Extra stream filters in the form {"field":"value",...}.
	filters, err := parseExtraFiltersJSON(s)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(filters))
	for i, f := range filters {
		if len(f.values) == 1 {
			result[i] = fmt.Sprintf("%q=%q", f.key, f.values[0])
		} else {
			orValues := make([]string, len(f.values))
			for j, v := range f.values {
				orValues[j] = regexp.QuoteMeta(v)
			}
			result[i] = fmt.Sprintf("%q=~%q", f.key, strings.Join(orValues, "|"))
		}
	}
	return logstorage.ParseFilter("{" + strings.Join(result, ",") + "}")
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
