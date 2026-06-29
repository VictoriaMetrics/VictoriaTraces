package tempo

import (
	"encoding/base64"
	"encoding/hex"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// hexIDToBase64 converts a hex-encoded trace/span ID (as stored in the span structs)
// into the base64 OTLP/JSON representation used by the Tempo trace-by-id JSON APIs.
//
// If the input is not valid hex, its raw bytes are base64-encoded, mirroring the
// fallback in the protobuf marshaller.
func hexIDToBase64(hexID string) string {
	if hexID == "" {
		return ""
	}
	b, err := hex.DecodeString(hexID)
	if err != nil {
		b = []byte(hexID)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// bytesToBase64 encodes raw bytes as base64 for OTLP/JSON bytesValue fields.
func bytesToBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

var spanKindNames = map[otelpb.SpanKind]string{
	0: "SPAN_KIND_UNSPECIFIED",
	1: "SPAN_KIND_INTERNAL",
	2: "SPAN_KIND_SERVER",
	3: "SPAN_KIND_CLIENT",
	4: "SPAN_KIND_PRODUCER",
	5: "SPAN_KIND_CONSUMER",
}

// spanKindName returns the OTLP/JSON enum name for a span kind.
func spanKindName(k otelpb.SpanKind) string {
	if n, ok := spanKindNames[k]; ok {
		return n
	}
	return "SPAN_KIND_UNSPECIFIED"
}

var statusCodeNames = map[otelpb.StatusCode]string{
	0: "STATUS_CODE_UNSET",
	1: "STATUS_CODE_OK",
	2: "STATUS_CODE_ERROR",
}

// statusCodeName returns the OTLP/JSON enum name for a status code.
func statusCodeName(c otelpb.StatusCode) string {
	if n, ok := statusCodeNames[c]; ok {
		return n
	}
	return "STATUS_CODE_UNSET"
}
