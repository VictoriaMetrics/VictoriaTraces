package pb

import "github.com/VictoriaMetrics/easyproto"

// Tempo Proto

type TempoTraceByIDResponse struct {
	Trace TempoTrace
}

// MarshalProtobuf marshals r to protobuf message, appends it to dst and returns the result.
func (tbir *TempoTraceByIDResponse) MarshalProtobuf(dst []byte) []byte {
	m := mp.Get()
	tbir.marshalProtobuf(m.MessageMarshaler())
	dst = m.Marshal(dst)
	mp.Put(m)
	return dst
}
func (tbir *TempoTraceByIDResponse) marshalProtobuf(mm *easyproto.MessageMarshaler) {
	//message TraceByIDResponse {
	//	Trace trace = 1;
	//	TraceByIDMetrics metrics = 2;
	//	PartialStatus status = 3;
	//	string message = 4;
	//}
	tbir.Trace.marshalProtobuf(mm.AppendMessage(1))
}

type TempoTrace struct {
	ResourceSpan []*ResourceSpans
}

func (t *TempoTrace) marshalProtobuf(mm *easyproto.MessageMarshaler) {
	//message Trace {
	//	repeated tempopb.trace.v1.ResourceSpans resourceSpans = 1;
	//}
	for _, rs := range t.ResourceSpan {
		rs.marshalProtobuf(mm.AppendMessage(1))
	}
}
