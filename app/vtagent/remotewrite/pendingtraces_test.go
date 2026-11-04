package remotewrite

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/encoding/zstd"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/persistentqueue"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// TestBatchingExportTraceServiceRequests verify if ExportTraceServiceRequest (or actually any protobuf message) can be batched directly
// by ByteBuffer.Write(msg1) and ByteBuffer.Write(msg2).
//
// See: https://github.com/protocolbuffers/protobuf/issues/9257#issuecomment-979447922
func TestBatchingExportTraceServiceRequests(t *testing.T) {
	newRequest := func(serviceName string) *otelpb.ExportTraceServiceRequest {
		return &otelpb.ExportTraceServiceRequest{
			ResourceSpans: []*otelpb.ResourceSpans{
				{
					Resource: otelpb.Resource{
						Attributes: []*otelpb.KeyValue{
							{Key: "service.name", Value: &otelpb.AnyValue{StringValue: &serviceName}},
						},
					},
				},
			},
		}
	}

	var wr writeRequest
	_, _ = wr.pendingData.Write(newRequest("svc1").MarshalProtobuf(nil))
	_, _ = wr.pendingData.Write(newRequest("svc2").MarshalProtobuf(nil))

	var req otelpb.ExportTraceServiceRequest
	if err := req.UnmarshalProtobuf(wr.pendingData.B); err != nil {
		t.Fatalf("cannot unmarshal concatenated requests: %s", err)
	}
	if len(req.ResourceSpans) != 2 {
		t.Fatalf("unexpected number of resource spans; got %d; want 2", len(req.ResourceSpans))
	}
	for i, want := range []string{"svc1", "svc2"} {
		got := *req.ResourceSpans[i].Resource.Attributes[0].Value.StringValue
		if got != want {
			t.Fatalf("unexpected service.name at resource spans #%d; got %q; want %q", i, got, want)
		}
	}
}

func TestPendingTracesAddBlockSize(t *testing.T) {
	defaultMaxBlockSize := maxUnpackedBlockSize.N
	defer func() {
		maxUnpackedBlockSize.N = defaultMaxBlockSize
	}()
	maxUnpackedBlockSize.N = 100

	f := func(reqSizes, blockSizesExpected []int) {
		t.Helper()

		fq := persistentqueue.MustOpenFastQueue(t.TempDir(), "test", 100, 0, false)
		defer fq.MustClose()
		pt := &pendingTraces{fq: fq}

		for _, n := range reqSizes {
			pt.add(make([]byte, n), 1)
		}
		pt.mustFlushLocked()

		var blockSizes []int
		for {
			block, ok := fq.MustReadInMemoryBlock(nil)
			if !ok {
				break
			}
			data, err := zstd.Decompress(nil, block)
			if err != nil {
				t.Fatalf("cannot decompress block: %s", err)
			}
			blockSizes = append(blockSizes, len(data))
		}
		if !reflect.DeepEqual(blockSizes, blockSizesExpected) {
			t.Fatalf("unexpected block sizes; got %v; want %v", blockSizes, blockSizesExpected)
		}
	}

	// requests fit a single block
	f([]int{30, 30, 40}, []int{100})
	f([]int{100}, []int{100})

	// the pending data is flushed before the request, which would exceed the max block size
	f([]int{60, 50}, []int{60, 50})
	f([]int{99, 2}, []int{99, 2})
	f([]int{100, 1}, []int{100, 1})
	f([]int{50, 50, 50}, []int{100, 50})
}

func TestPushOTLPTracesTooBigRequest(t *testing.T) {
	defaultMaxBlockSize := maxUnpackedBlockSize.N
	defer func() {
		maxUnpackedBlockSize.N = defaultMaxBlockSize
	}()
	maxUnpackedBlockSize.N = 100

	f := func(reqSize int, resultExpected bool) {
		t.Helper()
		err := PushOTLPTraces(make([]byte, reqSize), 1)
		if (err == nil) != resultExpected {
			t.Fatalf("unexpected result for %d bytes request; got error %v; want success %v", reqSize, err, resultExpected)
		}
	}

	f(100, true)
	f(101, false)
}

func TestPendingTracesDropTooBigCompressedBlock(t *testing.T) {
	defaultMaxBlockSize := maxUnpackedBlockSize.N
	defer func() {
		maxUnpackedBlockSize.N = defaultMaxBlockSize
	}()
	maxUnpackedBlockSize.N = 2 * persistentqueue.MaxBlockSize

	fq := persistentqueue.MustOpenFastQueue(t.TempDir(), "test", 100, 0, false)
	defer fq.MustClose()
	pt := &pendingTraces{fq: fq}

	// random data is incompressible, so the compressed block exceeds persistentqueue.MaxBlockSize
	req := make([]byte, persistentqueue.MaxBlockSize+1)
	r := rand.New(rand.NewSource(1))
	_, _ = r.Read(req)

	droppedSpansBefore := tooBigBlocksDroppedSpans.Get()
	pt.add(req, 10)
	pt.mustFlushLocked()

	if n := tooBigBlocksDroppedSpans.Get() - droppedSpansBefore; n != 10 {
		t.Fatalf("unexpected number of dropped spans; got %d; want 10", n)
	}
	if n := fq.GetPendingBytes(); n != 0 {
		t.Fatalf("unexpected pending bytes in the queue; got %d; want 0", n)
	}
	if n := len(pt.wr.pendingData.B); n != 0 {
		t.Fatalf("unexpected pending data size; got %d; want 0", n)
	}
}
