package opentelemetry

import (
	"fmt"
	"strconv"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/easyproto"

	"github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// the pushSpansHandler must store trace span with the given args.
//
// The handler must copy resource and attributes before returning,
// since the caller can change them, so they become invalid if not copied.
type pushSpansHandler func(timestamp int64, fields []logstorage.Field)

// decodeExportTraceServiceRequest parses an ExportTraceServiceRequest protobuf message from the src.
// The pushSpans callback function will be invoked at the end of the entire decoding process （most likely in decodeScopeSpans).
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/collector/trace/v1/trace_service.proto#L36
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/collector/trace/v1/trace_service.pb.go#L33
func decodeExportTraceServiceRequest(src []byte, pushSpans pushSpansHandler) (err error) {
	//message ExportTraceServiceRequest {
	//	repeated opentelemetry.proto.trace.v1.ResourceSpans resource_spans = 1;
	//}
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field in ExportTraceServiceRequest: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read resource spans data")
			}

			if err = decodeResourceSpans(data, pushSpans); err != nil {
				return fmt.Errorf("cannot decode resource span: %w", err)
			}
		}
	}
	return nil
}

// decodeResourceSpans parses a ResourceSpans protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L48
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L230
func decodeResourceSpans(src []byte, pushSpans pushSpansHandler) (err error) {
	//message ResourceSpans {
	//	opentelemetry.proto.resource.v1.Resource resource = 1;
	//	repeated ScopeSpans scope_spans = 2;
	//	string schema_url = 3;
	//}
	fb := getFmtBuffer()
	defer putFmtBuffer(fb)

	fs := logstorage.GetFields()
	defer func() {
		// Explicitly clear fs up to its capacity in order to free up
		// all the references to the original byte slice, so it could be freed by Go GC.
		fs.ClearUpToCapacity()
		logstorage.PutFields(fs)
	}()

	// Decode resource
	resourceData, ok, err := easyproto.GetMessageData(src, 1)
	if err != nil {
		return fmt.Errorf("cannot find Resource: %w", err)
	}
	if ok {
		if err = decodeResource(resourceData, fs, fb); err != nil {
			return fmt.Errorf("cannot decode Resource: %w", err)
		}
	}

	commonFieldsLen := len(fs.Fields)
	fbLen := len(fb.buf)

	// Decode scope_spans
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field: %w", err)
		}
		switch fc.FieldNum {
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read ScopeSpans data")
			}

			if err := decodeScopeSpans(data, fs, fb, pushSpans); err != nil {
				return fmt.Errorf("cannot decode ScopeSpans: %w", err)
			}

			fs.Fields = fs.Fields[:commonFieldsLen]
			fb.buf = fb.buf[:fbLen]
		}
	}

	return nil
}

// decodeResource parses a Resource protobuf message from src.
func decodeResource(src []byte, fs *logstorage.Fields, fb *fmtBuffer) (err error) {
	// message Resource {
	//   repeated KeyValue attributes = 1;
	// }
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field in Resource: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read Attributes data")
			}

			if err := decodeKeyValue(data, fs, fb, pb.ResourceAttrPrefix); err != nil {
				return fmt.Errorf("cannot decode Attributes: %w", err)
			}
		}
	}
	return nil
}

// decodeScopeSpans parses a Resource protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L68
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L308
func decodeScopeSpans(src []byte, fs *logstorage.Fields, fb *fmtBuffer, pushSpans pushSpansHandler) error {
	//message ScopeSpans {
	//	opentelemetry.proto.common.v1.InstrumentationScope scope = 1;
	//	repeated Span spans = 2;
	//	string schema_url = 3;
	//}
	scopeData, ok, err := easyproto.GetMessageData(src, 1)
	if err != nil {
		return fmt.Errorf("cannot read InstrumentationScope: %w", err)
	}
	if ok {
		if err = decodeInstrumentationScope(scopeData, fs, fb); err != nil {
			return fmt.Errorf("cannot decode InstrumentationScope: %w", err)
		}
	}

	commonFieldsLen := len(fs.Fields)
	fbLen := len(fb.buf)

	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field: %w", err)
		}
		switch fc.FieldNum {
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read Span data")
			}
			startTimeUnixNano, err := decodeSpan(data, fs, fb)
			if err != nil {
				return fmt.Errorf("cannot decode Span: %w", err)
			}
			pushSpans(int64(startTimeUnixNano), fs.Fields)
			fs.Fields = fs.Fields[:commonFieldsLen]
			fb.buf = fb.buf[:fbLen]
		}
	}
	return nil
}

// decodeInstrumentationScope parses a InstrumentationScope protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/common/v1/common.proto#L71
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/common/v1/common.pb.go#L340
func decodeInstrumentationScope(src []byte, fs *logstorage.Fields, fb *fmtBuffer) error {
	//message InstrumentationScope {
	//	string name = 1;
	//	string version = 2;
	//	repeated KeyValue attributes = 3;
	//	uint32 dropped_attributes_count = 4;
	//}
	nameStr, ok, err := easyproto.GetString(src, 1)
	if err != nil {
		return fmt.Errorf("cannot read name: %w", err)
	}
	if ok {
		fs.Add(pb.InstrumentationScopeName, nameStr)
	}

	versionStr, ok, err := easyproto.GetString(src, 2)
	if err != nil {
		return fmt.Errorf("cannot read version: %w", err)
	}
	if ok {
		fs.Add(pb.InstrumentationScopeVersion, versionStr)
	}

	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field: %w", err)
		}
		switch fc.FieldNum {
		case 3:
			attributesData, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read Attributes data")
			}
			if err := decodeKeyValue(attributesData, fs, fb, pb.InstrumentationScopeAttrPrefix); err != nil {
				return fmt.Errorf("cannot decode Attributes: %w", err)
			}
		}
	}

	return nil
}

// decodeSpan parses a Span protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L88
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L380
func decodeSpan(src []byte, fs *logstorage.Fields, fb *fmtBuffer) (startTimeUnixNano uint64, err error) {
	//message Span {
	//	bytes trace_id = 1;
	//	bytes span_id = 2;
	//	string trace_state = 3;
	//	bytes parent_span_id = 4;
	//	string name = 5;
	//	SpanKind kind = 6;
	//	fixed64 start_time_unix_nano = 7;
	//	fixed64 end_time_unix_nano = 8;
	//	repeated opentelemetry.proto.common.v1.KeyValue attributes = 9;
	//	uint32 dropped_attributes_count = 10;
	//	repeated Event events = 11;
	//	uint32 dropped_events_count = 12;
	//	repeated Link links = 13;
	//	uint32 dropped_links_count = 14;
	//	Status status = 15;
	//  fixed32 flags = 16;
	//}
	var (
		fc              easyproto.FieldContext
		ok              bool
		endTimeUnixNano uint64
		eventIdx        int
		linkIdx         int

		// special fields that must be appneded at the end of the fields slice
		// startTimeUnixNano uint64
		// endTimeUnixNano uint64
		traceID string
	)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return startTimeUnixNano, fmt.Errorf("cannot read next field in Span: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			traceIDBytes, ok := fc.Bytes()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span trace id")
			}
			traceID = fb.formatHex(traceIDBytes)
			// don't add to fs here. traceID field should be appended at the tail.
		case 2:
			spanID, ok := fc.Bytes()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span span id")
			}
			spanIDHex := fb.formatHex(spanID)
			fs.Add(pb.SpanIDField, spanIDHex)
		case 3:
			traceState, ok := fc.String()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span trace state")
			}
			fs.Add(pb.TraceStateField, traceState)
		case 4:
			parentSpanID, ok := fc.Bytes()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span parent span id")
			}
			parentSpanIDHex := fb.formatHex(parentSpanID)
			fs.Add(pb.ParentSpanIDField, parentSpanIDHex)
		case 5:
			spanName, ok := fc.String()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span name")
			}
			fs.Add(pb.NameField, spanName)
		case 6:
			kind, ok := fc.Int32()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span kind")
			}
			fs.Add(pb.KindField, strconv.FormatInt(int64(kind), 10))
		case 7:
			startTimeUnixNano, ok = fc.Fixed64()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span start timestamp")
			}
			// don't add to fs here. startTimeUnixNano field should be appended at the tail.
		case 8:
			endTimeUnixNano, ok = fc.Fixed64()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span end timestamp")
			}
			// don't add to fs here. endTimeUnixNano field should be appended at the tail.
		case 9:
			data, ok := fc.MessageData()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span attributes data")
			}
			if err = decodeKeyValue(data, fs, fb, pb.SpanAttrPrefixField); err != nil {
				return startTimeUnixNano, fmt.Errorf("cannot decode span attributes: %w", err)
			}
		case 10:
			droppedAttributesCount, ok := fc.Uint32()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span dropped attributes count")
			}
			fs.Add(pb.DroppedAttributesCountField, strconv.FormatUint(uint64(droppedAttributesCount), 10))
		case 11:
			data, ok := fc.MessageData()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span event data")
			}
			if err = decodeEvent(data, fs, fb, eventIdx); err != nil {
				return startTimeUnixNano, fmt.Errorf("cannot decode span event: %w", err)
			}
			eventIdx++
		case 12:
			droppedEventsCount, ok := fc.Uint32()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span dropped events count")
			}
			//s.DroppedEventsCount = droppedEventsCount
			fs.Add(pb.DroppedEventsCountField, strconv.FormatUint(uint64(droppedEventsCount), 10))
		case 13:
			data, ok := fc.MessageData()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span link data")
			}
			if err = decodeLink(data, fs, fb, linkIdx); err != nil {
				return startTimeUnixNano, fmt.Errorf("cannot decode span link: %w", err)
			}
			linkIdx++
		case 14:
			droppedLinksCount, ok := fc.Uint32()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span dropped links count")
			}
			//s.DroppedLinksCount = droppedLinksCount
			fs.Add(pb.DroppedLinksCountField, strconv.FormatUint(uint64(droppedLinksCount), 10))
		case 15:
			data, ok := fc.MessageData()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span status data")
			}
			if err = decodeStatus(data, fs, fb); err != nil {
				return startTimeUnixNano, fmt.Errorf("cannot decode span status: %w", err)
			}
		case 16:
			flags, ok := fc.Fixed32()
			if !ok {
				return startTimeUnixNano, fmt.Errorf("cannot read span flags")
			}
			fs.Add(pb.FlagsField, strconv.FormatUint(uint64(flags), 10))
		}
	}

	if endTimeUnixNano > 0 && startTimeUnixNano > 0 {
		fs.Add(pb.DurationField, strconv.FormatUint(endTimeUnixNano-startTimeUnixNano, 10))
	}

	// special fields that should be placed at the tail for faster lookup
	if startTimeUnixNano > 0 {
		fs.Add(pb.StartTimeUnixNanoField, strconv.FormatUint(startTimeUnixNano, 10))
	}
	if endTimeUnixNano > 0 {
		fs.Add(pb.EndTimeUnixNanoField, strconv.FormatUint(endTimeUnixNano, 10))
	}
	if traceID != "" {
		fs.Add(pb.TraceIDField, traceID)
	}
	return startTimeUnixNano, nil
}

// decodeEvent parses an Event protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L222
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L613
func decodeEvent(src []byte, fs *logstorage.Fields, fb *fmtBuffer, eventIdx int) (err error) {
	//message Event {
	//	fixed64 time_unix_nano = 1;
	//	string name = 2;
	//	repeated opentelemetry.proto.common.v1.KeyValue attributes = 3;
	//	uint32 dropped_attributes_count = 4;
	//}
	var fc easyproto.FieldContext
	eventFieldSuffix := ":" + strconv.Itoa(eventIdx)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field in Status: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			ts, ok := fc.Fixed64()
			if !ok {
				return fmt.Errorf("cannot read span event timestamp")
			}
			fs.Add(pb.EventPrefix+pb.EventTimeUnixNanoField+eventFieldSuffix, strconv.FormatUint(ts, 10))
		case 2:
			name, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read span event name")
			}
			fs.Add(pb.EventPrefix+pb.EventNameField+eventFieldSuffix, name)
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read span event attributes data")
			}
			if err = decodeKeyValueWithPrefixSuffix(data, fs, fb, "", pb.EventPrefix+pb.EventAttrPrefix, eventFieldSuffix); err != nil {
				return fmt.Errorf("cannot decode span event attributes: %w", err)
			}
		case 4:
			droppedAttributesCount, ok := fc.Uint32()
			if !ok {
				return fmt.Errorf("cannot read span event dropped attributes count")
			}
			fs.Add(pb.EventPrefix+pb.EventDroppedAttributesCountField+eventFieldSuffix, strconv.FormatUint(uint64(droppedAttributesCount), 10))
		}
	}
	return nil
}

// decodeLink parses a Link protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L251
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L693
func decodeLink(src []byte, fs *logstorage.Fields, fb *fmtBuffer, linkIdx int) (err error) {
	//message Link {
	//	bytes trace_id = 1;
	//	bytes span_id = 2;
	//	string trace_state = 3;
	//	repeated opentelemetry.proto.common.v1.KeyValue attributes = 4;
	//	uint32 dropped_attributes_count = 5;
	//	fixed32 flags = 6;
	//}
	var fc easyproto.FieldContext
	linkFieldSuffix := ":" + strconv.Itoa(linkIdx)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field in Status: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			traceID, ok := fc.Bytes()
			if !ok {
				return fmt.Errorf("cannot read span link trace id")
			}
			traceIDHex := fb.formatHex(traceID)
			fs.Add(pb.LinkPrefix+pb.LinkTraceIDField+linkFieldSuffix, traceIDHex)
		case 2:
			spanID, ok := fc.Bytes()
			if !ok {
				return fmt.Errorf("cannot read span link span id")
			}
			spanIDHex := fb.formatHex(spanID)
			fs.Add(pb.LinkPrefix+pb.LinkSpanIDField+linkFieldSuffix, spanIDHex)
		case 3:
			traceState, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read span link trace state")
			}
			fs.Add(pb.LinkPrefix+pb.LinkTraceStateField+linkFieldSuffix, traceState)
		case 4:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read aspan link ttributes data")
			}
			if err = decodeKeyValueWithPrefixSuffix(data, fs, fb, "", pb.LinkPrefix+pb.LinkAttrPrefix, linkFieldSuffix); err != nil {
				return fmt.Errorf("cannot decode span link attributes: %w", err)
			}
		case 5:
			droppedAttributesCount, ok := fc.Uint32()
			if !ok {
				return fmt.Errorf("cannot read span link dropped attributes count")
			}
			//sl.DroppedAttributesCount = droppedAttributesCount
			fs.Add(pb.LinkPrefix+pb.LinkDroppedAttributesCountField+linkFieldSuffix, strconv.FormatUint(uint64(droppedAttributesCount), 10))
		case 6:
			flags, ok := fc.Fixed32()
			if !ok {
				return fmt.Errorf("cannot read span link flags")
			}
			fs.Add(pb.LinkPrefix+pb.LinkFlagsField+linkFieldSuffix, strconv.FormatUint(uint64(flags), 10))

		}
	}
	return nil
}

// decodeStatus parses a Status protobuf message from src.
//
// https://github.com/open-telemetry/opentelemetry-proto/blob/v1.5.0/opentelemetry/proto/trace/v1/trace.proto#L306
// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.124.0/pdata/internal/data/protogen/trace/v1/trace.pb.go#L791
func decodeStatus(src []byte, fs *logstorage.Fields, fb *fmtBuffer) (err error) {
	//message Status {
	//	reserved 1;
	//	string message = 2;
	//	StatusCode code = 3;
	//}
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field in Status: %w", err)
		}
		switch fc.FieldNum {
		case 2:
			message, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read status message")
			}
			fs.Add(pb.StatusMessageField, message)
		case 3:
			code, ok := fc.Int32()
			if !ok {
				return fmt.Errorf("cannot read status code")
			}
			fs.Add(pb.StatusCodeField, strconv.FormatInt(int64(code), 10))
		}
	}
	return nil
}

// decodeKeyValueWithPrefixSuffix parses a KeyValue message from src.
func decodeKeyValueWithPrefixSuffix(src []byte, fs *logstorage.Fields, fb *fmtBuffer, parentField, prefix, suffix string) error {
	// message KeyValue {
	//   string key = 1;
	//   AnyValue value = 2;
	// }

	// Decode key
	key, ok, err := easyproto.GetString(src, 1)
	if err != nil {
		return fmt.Errorf("cannot find Key in KeyValue: %w", err)
	}
	if !ok {
		// Key is missing, skip it.
		// See https://github.com/VictoriaMetrics/VictoriaLogs/issues/869#issuecomment-3631307996
		return nil
	}
	fieldName := fb.formatSubFieldName(parentField, key)

	// Decode value
	valueData, ok, err := easyproto.GetMessageData(src, 2)
	if err != nil {
		return fmt.Errorf("cannot find Value in KeyValue: %w", err)
	}
	if !ok {
		// Value is null, skip it.
		return nil
	}

	if err := decodeAnyValue(valueData, fs, fb, fieldName, prefix, suffix); err != nil {
		return fmt.Errorf("cannot decode AnyValue: %w", err)
	}
	return nil
}

func decodeKeyValue(src []byte, fs *logstorage.Fields, fb *fmtBuffer, fieldNamePrefix string) error {
	return decodeKeyValueWithPrefixSuffix(src, fs, fb, "", fieldNamePrefix, "")
}

func decodeAnyValue(src []byte, fs *logstorage.Fields, fb *fmtBuffer, fieldName, prefix, suffix string) (err error) {
	// message AnyValue {
	//   oneof value {
	//     string string_value = 1;
	//     bool bool_value = 2;
	//     int64 int_value = 3;
	//     double double_value = 4;
	//     ArrayValue array_value = 5;
	//     KeyValueList kvlist_value = 6;
	//     bytes bytes_value = 7;
	//   }
	// }

	fullFieldName := prefix + fieldName + suffix
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			stringValue, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read StringValue")
			}
			if stringValue == "" {
				fs.Add(fullFieldName, "-")
				continue
			}
			fs.Add(fullFieldName, stringValue)
		case 2:
			boolValue, ok := fc.Bool()
			if !ok {
				return fmt.Errorf("cannot read BoolValue")
			}
			boolValueStr := strconv.FormatBool(boolValue)
			fs.Add(fullFieldName, boolValueStr)
		case 3:
			intValue, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read IntValue")
			}
			intValueStr := fb.formatInt(intValue)
			fs.Add(fullFieldName, intValueStr)
		case 4:
			doubleValue, ok := fc.Double()
			if !ok {
				return fmt.Errorf("cannot read DoubleValue")
			}
			doubleValueStr := fb.formatFloat(doubleValue)
			fs.Add(fullFieldName, doubleValueStr)
		case 5:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read ArrayValue")
			}

			a := jsonArenaPool.Get()
			// Encode arrays as JSON to match the behavior of /insert/jsonline
			arr, err := decodeArrayValueToJSON(data, a, fb)
			if err != nil {
				jsonArenaPool.Put(a)
				return fmt.Errorf("cannot decode ArrayValue: %w", err)
			}
			encodedArr := fb.encodeJSONValue(arr)
			jsonArenaPool.Put(a)

			fs.Add(fullFieldName, encodedArr)
		case 6:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read KeyValueList")
			}
			if err := decodeKeyValueList(data, fs, fb, fieldName, prefix, suffix); err != nil {
				return fmt.Errorf("cannot decode KeyValueList: %w", err)
			}
		case 7:
			bytesValue, ok := fc.Bytes()
			if !ok {
				return fmt.Errorf("cannot read BytesValue")
			}
			v := fb.formatBase64(bytesValue)
			fs.Add(fullFieldName, v)
		}
	}
	return nil
}

func decodeKeyValueList(src []byte, fs *logstorage.Fields, fb *fmtBuffer, parentField, prefix, suffix string) (err error) {
	// message KeyValueList {
	//   repeated KeyValue values = 1;
	// }

	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read the next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read KeyValue data")
			}
			if err := decodeKeyValueWithPrefixSuffix(data, fs, fb, parentField, prefix, suffix); err != nil {
				return fmt.Errorf("cannot decode KeyValue: %w", err)
			}
		}
	}
	return nil
}
