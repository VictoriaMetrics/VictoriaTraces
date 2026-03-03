package main

import (
	"encoding/binary"
	"net/http"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/encoding/zstd"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"

	_ "net/http/pprof"
)

var (
	maxRequestSize    = flagutil.NewBytes("opentelemetry.sampling.maxRequestSize", 16*1024*1024, "The maximum size in bytes of a single OpenTelemetry trace sampling request.")
	slowTraceDuration = flagutil.NewExtendedDuration("opentelemetry.sampling.slowTraceDuration", "5s", "Traces that last longer than this duration will be sampled as slow traces.")
)

// local test
var agentAddrs = []string{
	"http://127.0.0.1:10499/api/v1/remotesampling_decision",
}

type SamplingTrace struct {
	TraceID    [16]byte `json:"trace_id"`
	StartTime  uint64   `json:"start_time"`
	EndTime    uint64   `json:"end_time"`
	StatusCode int32    `json:"status_code"`

	Sampled bool
}

var (
	mu                  = sync.Mutex{}
	waitingTraceMapCur  = map[[16]byte]SamplingTrace{}
	waitingTraceMapPrev = map[[16]byte]SamplingTrace{}
)

func startBufCleaner(handlingFunc func([][16]byte), interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var tidList [][16]byte
				mu.Lock()
				for _, trace := range waitingTraceMapPrev {
					if trace.Sampled {
						tidList = append(tidList, trace.TraceID)
						continue
					}

					duration := trace.EndTime - trace.StartTime
					if duration > uint64((*slowTraceDuration).Duration().Nanoseconds()) {
						logger.Infof("sampled for %d ms", duration/uint64(time.Millisecond))
						tidList = append(tidList, trace.TraceID)
						continue
					}

					if trace.StatusCode == 2 {
						tidList = append(tidList, trace.TraceID)
						continue
					}
				}

				prevLen := len(waitingTraceMapPrev)
				// drop prev one and create a new one
				waitingTraceMapPrev = make(map[[16]byte]SamplingTrace, prevLen)

				// rotate the cur one to prev, and prev as cur.
				waitingTraceMapCur, waitingTraceMapPrev = waitingTraceMapPrev, waitingTraceMapCur

				mu.Unlock()

				handlingFunc(tidList)
			}
		}
	}()
}

func fanoutDecisions(tidList [][16]byte) {
	if len(tidList) == 0 {
		return
	}
	logger.Infof("fanout sampled decision for %d traces", len(tidList))

	decisionBytes := make([]byte, 0, 16*len(tidList))
	for _, trace := range tidList {
		decisionBytes = append(decisionBytes, trace[:]...)
	}

	bb := zstdBufPool.Get()
	defer zstdBufPool.Put(bb)

	bb.B = zstd.CompressLevel(bb.B[:0], decisionBytes, 1)

	for _, addr := range agentAddrs {
		req, err := http.NewRequest(http.MethodPost, addr, bb.NewReader())
		if err != nil {
			logger.Panicf("BUG: cannot create a new HTTP request to %q: %s", addr, err)
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Encoding", "zstd")
		req.Header.Set("User-Agent", "retroactivesamplingserver/0.1")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Errorf("cannot post agent to %s: %s", addr, err)
			continue
		}
		resp.Body.Close()
	}
}

var zstdBufPool bytesutil.ByteBufferPool

func main() {
	rh := func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/insert/sampling":
			samplingRequestHandler(w, r)
			return true
		}
		return false
	}
	go httpserver.Serve([]string{"0.0.0.0:10430"}, rh, httpserver.ServeOptions{})
	startBufCleaner(fanoutDecisions, 15*time.Second)
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
}

var (
	decisionSampled    uint8 = 1
	decisionNotSampled uint8 = 0
)

func samplingRequestHandler(w http.ResponseWriter, r *http.Request) {
	encoding := r.Header.Get("Content-Encoding")
	err := protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, handleSamplingRequestData)
	if err != nil {
		logger.Errorf("cannot unmarshal HTTP request: %s", err)
		httpserver.Errorf(w, r, "cannot unmarshal HTTP request: %s", err)
		return
	}

	w.WriteHeader(200)
}

// handleSamplingRequestData handle the decompressed request and set them to the cache.
func handleSamplingRequestData(data []byte) error {
	for len(data) > 0 {
		// samplingBuf is the metadata extracted from each span.
		// it should be marshaled into:
		// [<decision>|<traceID>|<startTime>|<endTime>|<statusCode>] in bytes request.
		// [1 byte    |16 bytes |8 bytes    |8 bytes  |1 byte      ] = 34 bytes
		// If decision is true, then it contains only [<decision>|<traceID>] as the rest are not useful anymore.
		if data[0] == decisionSampled {
			if len(data) < (1 + 16) {
				logger.Errorf("got %d bytes when we want at least 17 bytes for sampled trace", len(data))
				break
			}

			var traceID [16]byte
			copy(traceID[:], data[1:1+16])

			data = data[1+16:] // [17:]

			var (
				trace SamplingTrace
				ok    bool
			)

			mu.Lock()
			if trace, ok = waitingTraceMapCur[traceID]; ok {
				trace.Sampled = true
				waitingTraceMapCur[traceID] = trace
			} else if trace, ok = waitingTraceMapPrev[traceID]; ok {
				trace.Sampled = true
				waitingTraceMapPrev[traceID] = trace
			} else {
				trace = SamplingTrace{
					TraceID: traceID,
					Sampled: true,
				}
				waitingTraceMapCur[traceID] = trace
			}
			mu.Unlock()
			continue
		}

		// slow path, need to record.
		if len(data) < 34 {
			logger.Errorf("got %d bytes when we want at least 34 bytes for unsampled trace", len(data))
			break
		}

		// not sampled, add to buf and wait
		var traceID [16]byte
		copy(traceID[:], data[1:1+16])
		startTimeByte := data[1+16 : 1+16+8]
		endTimeByte := data[1+16+8 : 1+16+8+8]
		statusCodeByte := data[1+16+8+8 : 1+16+8+8+1]
		data = data[1+16+8+8+1:] // [34:]

		startTime := binary.BigEndian.Uint64(startTimeByte)
		endTime := binary.BigEndian.Uint64(endTimeByte)
		stausCode := int32(statusCodeByte[0])

		// find the same trace from cache
		var (
			trace SamplingTrace
			ok    bool
		)
		mu.Lock()
		if trace, ok = waitingTraceMapCur[traceID]; ok {
			if trace.Sampled {
				continue
			}
			trace.EndTime = max(trace.EndTime, endTime)
			trace.StartTime = min(trace.StartTime, startTime)
			if stausCode == 2 {
				trace.StatusCode = 2
			}
			waitingTraceMapCur[traceID] = trace
		} else if trace, ok = waitingTraceMapPrev[traceID]; ok {
			if trace.Sampled {
				continue
			}
			trace.EndTime = max(trace.EndTime, endTime)
			trace.StartTime = min(trace.StartTime, startTime)
			if stausCode == 2 {
				trace.StatusCode = 2
			}
			waitingTraceMapPrev[traceID] = trace
		} else {
			trace = SamplingTrace{
				TraceID:    traceID,
				StartTime:  startTime,
				EndTime:    endTime,
				StatusCode: stausCode,

				Sampled: false,
			}
			waitingTraceMapCur[traceID] = trace
		}

		mu.Unlock()
	}

	return nil
}
