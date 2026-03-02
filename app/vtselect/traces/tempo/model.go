package tempo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/tracecommon"
	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

func rowsToResourceSpans(rows []*tracecommon.Row) ([]*otelpb.ResourceSpans, error) {
	// group data by service.name first
	result := make([]*otelpb.ResourceSpans, 0, len(rows))
	for _, row := range rows {
		scope := otelpb.InstrumentationScope{}
		sp := &otelpb.Span{}
		resource := otelpb.Resource{
			Attributes: make([]*otelpb.KeyValue, 0),
		}

		spanEventMap := make(map[string]*otelpb.SpanEvent) // idx -> *Log
		spanLinkMap := make(map[string]*otelpb.SpanLink)   // idx -> *SpanRef

		rs := &otelpb.ResourceSpans{
			Resource: resource,
			ScopeSpans: []*otelpb.ScopeSpans{
				{
					Scope: scope,
					Spans: []*otelpb.Span{
						sp,
					},
				},
			},
		}

		for _, field := range row.Fields {
			switch field.Name {
			case "_stream":
				// no-op
			case otelpb.TraceIDField:
				sp.TraceID = field.Value
			case otelpb.SpanIDField:
				sp.SpanID = field.Value
			case otelpb.ParentSpanIDField:
				sp.ParentSpanID = field.Value
			case otelpb.NameField:
				sp.Name = field.Value
			case otelpb.KindField:
				v, err := strconv.ParseInt(field.Value, 10, 32)
				if err != nil {
					return nil, err
				}
				sp.Kind = otelpb.SpanKind(v)
			case otelpb.FlagsField:
				// todo trace does not contain "flag" in result
				flagU64, err := strconv.ParseUint(field.Value, 10, 32)
				if err != nil {
					return nil, err
				}
				sp.Flags = uint32(flagU64)
			case otelpb.StartTimeUnixNanoField:
				unixNano, err := strconv.ParseInt(field.Value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid start_time_unix_nano field: %s", err)
				}
				sp.StartTimeUnixNano = uint64(unixNano)
			case otelpb.EndTimeUnixNanoField:
				unixNano, err := strconv.ParseInt(field.Value, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid end_time_unix_nano field: %s", err)
				}
				sp.EndTimeUnixNano = uint64(unixNano)
			case otelpb.DurationField:

			case otelpb.StatusCodeField:
				statusCode, err := strconv.ParseInt(field.Value, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid status_code field: %s", err)
				}
				sp.Status.Code = otelpb.StatusCode(statusCode)
			case otelpb.StatusMessageField:
				sp.Status.Message = field.Value
			case otelpb.TraceStateField:
				sp.TraceState = field.Value
			// resource level fields
			case otelpb.ResourceAttrServiceName:
				v := strings.Clone(field.Value)
				resource.Attributes = append(resource.Attributes, &otelpb.KeyValue{
					Key: "service.name",
					Value: &otelpb.AnyValue{
						StringValue: &v,
					},
				})
			// scope level fields
			case otelpb.InstrumentationScopeName:
				scope.Name = field.Value
			case otelpb.InstrumentationScopeVersion:
				scope.Version = field.Value
			default:
				v := strings.Clone(field.Value)
				if strings.HasPrefix(field.Name, otelpb.ResourceAttrPrefix) { // resource attributes
					resource.Attributes = append(resource.Attributes, &otelpb.KeyValue{
						Key: strings.TrimPrefix(field.Name, otelpb.ResourceAttrPrefix),
						Value: &otelpb.AnyValue{
							StringValue: &v,
						},
					})
				} else if strings.HasPrefix(field.Name, otelpb.SpanAttrPrefixField) { // span attributes
					sp.Attributes = append(sp.Attributes, &otelpb.KeyValue{
						Key: strings.TrimPrefix(field.Name, otelpb.SpanAttrPrefixField),
						Value: &otelpb.AnyValue{
							StringValue: &v,
						},
					})
				} else if strings.HasPrefix(field.Name, otelpb.InstrumentationScopeAttrPrefix) { // instrumentation scope attributes
					scope.Attributes = append(scope.Attributes, &otelpb.KeyValue{
						Key: strings.TrimPrefix(field.Name, otelpb.InstrumentationScopeAttrPrefix),
						Value: &otelpb.AnyValue{
							StringValue: &v,
						},
					})
				} else if strings.HasPrefix(field.Name, otelpb.EventPrefix) { // event list
					fieldName, idx := extraAttributeNameAndIndex(strings.TrimPrefix(field.Name, otelpb.EventPrefix))
					if idx == "" {
						return nil, fmt.Errorf("invalid event field: %s", field.Name)
					}
					if _, ok := spanEventMap[idx]; !ok {
						spanEventMap[idx] = &otelpb.SpanEvent{}
					}
					event := spanEventMap[idx]
					switch fieldName {
					case otelpb.EventTimeUnixNanoField:
						unixNano, _ := strconv.ParseInt(field.Value, 10, 64)
						event.TimeUnixNano = uint64(unixNano)
					case otelpb.EventNameField:
						event.Name = field.Value
					case otelpb.EventDroppedAttributesCountField:
						//no need to display
						//lg.Fields = append(lg.Fields, KeyValue{Key: fieldName, VStr: field.Value})
					default:
						event.Attributes = append(event.Attributes, &otelpb.KeyValue{
							Key: strings.TrimPrefix(fieldName, otelpb.EventAttrPrefix),
							Value: &otelpb.AnyValue{
								StringValue: &v,
							},
						})
					}
				} else if strings.HasPrefix(field.Name, otelpb.LinkPrefix) { // link list
					fieldName, idx := extraAttributeNameAndIndex(strings.TrimPrefix(field.Name, otelpb.LinkPrefix))
					if idx == "" {
						return nil, fmt.Errorf("invalid link field: %s", field.Name)
					}
					if _, ok := spanLinkMap[idx]; !ok {
						spanLinkMap[idx] = &otelpb.SpanLink{}
					}
					spanLink := spanLinkMap[idx]
					switch fieldName {
					case otelpb.LinkTraceIDField:
						spanLink.TraceID = field.Value
					case otelpb.LinkSpanIDField:
						spanLink.SpanID = field.Value
					case otelpb.LinkTraceStateField:
						spanLink.TraceState = field.Value
					case otelpb.LinkFlagsField:
						intv, err := strconv.ParseInt(field.Value, 10, 32)
						if err != nil {
							return nil, fmt.Errorf("invalid link flags: %s", err)
						}
						spanLink.Flags = uint32(intv)
					case otelpb.LinkDroppedAttributesCountField:
						// no need to display
					default:
						spanLink.Attributes = append(spanLink.Attributes, &otelpb.KeyValue{
							Key: strings.TrimPrefix(fieldName, otelpb.LinkAttrPrefix),
							Value: &otelpb.AnyValue{
								StringValue: &v,
							},
						})
					}
				}
			}
		}

		for i := 0; i < len(spanLinkMap); i++ {
			idx := strconv.Itoa(i)
			sp.Links = append(sp.Links, spanLinkMap[idx])
		}
		for i := 0; i < len(spanEventMap); i++ {
			idx := strconv.Itoa(i)
			sp.Events = append(sp.Events, spanEventMap[idx])
		}

		rs.Resource = resource
		rs.ScopeSpans[0].Scope = scope

		result = append(result, rs)
	}
	return result, nil
}

func extraAttributeNameAndIndex(input string) (string, string) {
	splitIdx := strings.LastIndex(input, ":")
	if splitIdx == -1 {
		return input, ""
	}
	idx := input[splitIdx+1:]
	if _, err := strconv.Atoi(idx); err != nil {
		return input, ""
	}
	return input[:splitIdx], idx
}
