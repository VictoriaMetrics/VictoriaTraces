package pb

// internal_fields.go contains the stream names/values, field names/values that VictoriaTraces required/generated.
//
// They're NOT part of the OpenTelemetry standard.

// TraceID index stream and fields
const (
	TraceIDIndexStreamName     = "trace_id_idx_stream"
	TraceIDIndexFieldName      = "trace_id_idx"
	TraceIDIndexPartitionCount = uint64(1024)
)

// service graph stream and fields
const (
	ServiceGraphStreamName         = "trace_service_graph_stream"
	ServiceGraphParentFieldName    = "parent"
	ServiceGraphChildFieldName     = "child"
	ServiceGraphCallCountFieldName = "callCount"
)
