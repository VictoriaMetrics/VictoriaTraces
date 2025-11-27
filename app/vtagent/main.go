package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/protoparser/protoparserutil"
)

var (
	maxRequestSize = flagutil.NewBytes("opentelemetry.sampling.maxRequestSize", 16*1024*1024, "The maximum size in bytes of a single OpenTelemetry trace sampling request.")
)

var agentAddrs = []string{
	"http://10.140.0.2:10429/api/v1/remotesampling_decision",
	"http://10.140.15.234:10429/api/v1/remotesampling_decision",
	"http://10.140.0.6:10429/api/v1/remotesampling_decision",
}

// local test
//var agentAddrs = []string{
//	"http://127.0.0.1:10429/api/v1/remotesampling_decision",
//}

type SamplingRequest struct {
	SamplingTraceList []*SamplingTrace `json:"sampling_trace_list"`
}

type SamplingTrace struct {
	TraceID    string `json:"trace_id"`
	StartTime  uint64 `json:"start_time"`
	EndTime    uint64 `json:"end_time"`
	StatusCode int32  `json:"status_code"`
}

type SamplingDecision struct {
	TraceIDList []string `json:"trace_id_list"`
}

var (
	waitingTraceMapCur  = sync.Map{}
	waitingTraceMapPrev = sync.Map{}
)

func startBufCleaner() {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var tidList []string
				waitingTraceMapPrev.Range(func(k, v interface{}) bool {
					trace := v.(*SamplingTrace)
					duration := trace.EndTime - trace.StartTime
					if duration > uint64(2000*time.Millisecond) && duration < uint64(2100*time.Millisecond) {
						tidList = append(tidList, trace.TraceID)
						return true
					}

					if trace.StatusCode == 2 {
						tidList = append(tidList, trace.TraceID)
						return true
					}

					if rand.Intn(100) < 1 {
						tidList = append(tidList, trace.TraceID)
						return true
					}

					return true
				})
				waitingTraceMapPrev.Clear()
				waitingTraceMapCur, waitingTraceMapPrev = waitingTraceMapPrev, waitingTraceMapCur

				if len(tidList) == 0 {
					continue
				}
				logger.Infof("fanout sampled decision for %d traces", len(tidList))

				go func() {
					sd := SamplingDecision{
						TraceIDList: tidList,
					}
					b, err := json.Marshal(&sd)
					if err != nil {
						logger.Errorf("cannot marshal SamplingDecision: %s", err)
						return
					}
					for _, addr := range agentAddrs {
						resp, err := http.Post(addr, "application/json", bytes.NewBuffer(b))
						if err != nil {
							logger.Errorf("cannot post agent to %s: %s", addr, err)
							continue
						}
						resp.Body.Close()
					}
				}()
			}
		}
	}()
}

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
	startBufCleaner()
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
}

func samplingRequestHandler(w http.ResponseWriter, r *http.Request) {
	encoding := r.Header.Get("Content-Encoding")
	err := protoparserutil.ReadUncompressedData(r.Body, encoding, maxRequestSize, func(data []byte) error {
		var req SamplingRequest

		if err := json.Unmarshal(data, &req); err != nil {
			logger.Errorf("cannot unmarshal HTTP request: %s", err)
			return err
		}

		for i := range req.SamplingTraceList {
			if req.SamplingTraceList[i] != nil {
				var trace *SamplingTrace
				if value, ok := waitingTraceMapCur.Load(req.SamplingTraceList[i].TraceID); ok {
					trace = value.(*SamplingTrace)
				} else if value, ok = waitingTraceMapPrev.Load(req.SamplingTraceList[i].TraceID); ok {
					trace = value.(*SamplingTrace)
				} else {
					trace = &SamplingTrace{
						TraceID:    req.SamplingTraceList[i].TraceID,
						StartTime:  req.SamplingTraceList[i].StartTime,
						EndTime:    req.SamplingTraceList[i].EndTime,
						StatusCode: req.SamplingTraceList[i].StatusCode,
					}
					waitingTraceMapCur.Store(req.SamplingTraceList[i].TraceID, trace)
					continue
				}

				trace.EndTime = max(trace.EndTime, req.SamplingTraceList[i].EndTime)
				trace.StartTime = min(trace.StartTime, req.SamplingTraceList[i].StartTime)
				if req.SamplingTraceList[i].StatusCode == 2 {
					trace.StatusCode = 2
				}
				//durationNano := trace.EndTime - trace.StartTime
				//
				//if durationNano > uint64(35000*time.Millisecond) {
				//	//if trace.StatusCode == 2 {
				//	//rand.Intn(100) < 1 {
				//	sampledTraceMap.Store(req.SamplingTraceList[i].TraceID, struct{}{})
				//}
			}

		}

		return nil
	})
	if err != nil {
		logger.Errorf("cannot unmarshal HTTP request: %s", err)
		httpserver.Errorf(w, r, "cannot unmarshal HTTP request: %s", err)
		return
	}

	w.WriteHeader(200)
}
