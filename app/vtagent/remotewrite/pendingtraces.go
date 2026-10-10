package remotewrite

import (
	"flag"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/encoding/zstd"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fasttime"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/persistentqueue"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/timeutil"
	"github.com/VictoriaMetrics/metrics"
)

var (
	maxUnpackedBlockSize = flagutil.NewBytes("remoteWrite.maxBlockSize", 8*1024*1024, "The maximum block size to send to remote storage. Bigger blocks may improve performance at the cost of the increased memory usage.")
	flushInterval        = flag.Duration("remoteWrite.flushInterval", time.Second, "Interval for flushing the data to remote storage. "+
		"This option takes effect only when less than 2MB of data per second are pushed to -remoteWrite.url")
)

type pendingTraces struct {
	lastFlushTime atomic.Uint64

	// The queue to send blocks to.
	fq *persistentqueue.FastQueue

	// mu protects wr
	mu sync.Mutex
	wr writeRequest

	stopCh            chan struct{}
	periodicFlusherWG sync.WaitGroup
}

func newPendingTraces(fq *persistentqueue.FastQueue) *pendingTraces {
	pt := &pendingTraces{
		fq:     fq,
		stopCh: make(chan struct{}),
	}

	pt.periodicFlusherWG.Go(pt.periodicFlusher)

	return pt
}

// add adds the OTLP ExportTraceServiceRequest protobuf message req with spansCount spans to pt.
func (pt *pendingTraces) add(req []byte, spansCount int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Concatenation of serialized ExportTraceServiceRequest messages is a valid ExportTraceServiceRequest message
	// containing resource_spans from all the concatenated messages, since resource_spans is a repeated field.
	// See https://protobuf.dev/programming-guides/encoding/#last-one-wins
	_, _ = pt.wr.pendingData.Write(req)
	pt.wr.pendingSpansCount += int64(spansCount)
	if len(pt.wr.pendingData.B) > maxUnpackedBlockSize.IntN() {
		pt.mustFlushLocked()
	}
}

func (pt *pendingTraces) mustFlushLocked() {
	pt.lastFlushTime.Store(fasttime.UnixTimestamp())
	pt.wr.push(func(b []byte) {
		if !pt.fq.TryWriteBlock(b) {
			logger.Fatalf("BUG: TryWriteBlock cannot return false")
		}
	})
}

func (pt *pendingTraces) periodicFlusher() {
	flushSeconds := int64(flushInterval.Seconds())
	if flushSeconds <= 0 {
		flushSeconds = 1
	}
	d := timeutil.AddJitterToDuration(*flushInterval)
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-pt.stopCh:
			pt.mu.Lock()
			pt.mustFlushOnStop()
			pt.mu.Unlock()
			return
		case <-ticker.C:
			if fasttime.UnixTimestamp()-pt.lastFlushTime.Load() < uint64(flushSeconds) {
				continue
			}
		}
		pt.mu.Lock()
		pt.mustFlushLocked()
		pt.mu.Unlock()
	}
}

// mustFlushOnStop force pushes wr data
//
// This is needed in order to properly save in-memory data to persistent queue on graceful shutdown.
func (pt *pendingTraces) mustFlushOnStop() {
	pt.wr.push(pt.fq.MustWriteBlockIgnoreDisabledPQ)
}

func (pt *pendingTraces) mustStop() {
	close(pt.stopCh)
	pt.periodicFlusherWG.Wait()
}

type writeRequest struct {
	// pendingData contains concatenated OTLP ExportTraceServiceRequest protobuf messages.
	pendingData       bytesutil.ByteBuffer
	pendingSpansCount int64
}

func (wr *writeRequest) push(pushBlock func([]byte)) {
	if len(wr.pendingData.B) == 0 {
		return
	}
	b := wr.pendingData.B

	zb := compressBufPool.Get()
	defer compressBufPool.Put(zb)

	zb.B = zstd.CompressLevel(zb.B[:0], b, 1)
	pushBlock(zb.B)

	blockSizeBytes.Update(float64(len(zb.B)))
	blockSizeSpans.Update(float64(wr.pendingSpansCount))

	wr.pendingData.Reset()
	wr.pendingSpansCount = 0
}

var (
	blockSizeBytes = metrics.NewHistogram(`vtagent_remotewrite_block_size_bytes`)
	blockSizeSpans = metrics.NewHistogram(`vtagent_remotewrite_block_size_spans`)
)

var compressBufPool bytesutil.ByteBufferPool
