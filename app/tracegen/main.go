package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"golang.org/x/time/rate"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var requestBodyList = make([][]byte, 0, 101)

func main() {
	spanRate := flag.Int("rate", 10000, "spans per second.")
	addr := flag.String("addr", "", "otlp trace export endpoint.")
	flag.Parse()
	if _, err := url.Parse(*addr); err != nil {
		panic(fmt.Sprintf("invalid otlp trace export endpoint: %v", err))
	}
	for i := 0; i <= 100; i++ {
		dat, err := os.ReadFile(fmt.Sprintf("%d.bin", i))
		if err != nil {
			panic(fmt.Sprintf("cannot read file %d: %v", i, err))
		}
		requestBodyList = append(requestBodyList, dat)
	}
	limiter := rate.NewLimiter(rate.Limit(*spanRate), *spanRate)
	for {
		traceIDMap := make(map[string]string)
		once := sync.Once{}
		timeOffset := uint64(0)
		for i := range requestBodyList {
			data := requestBodyList[i]
			var req otelpb.ExportTraceServiceRequest
			if err := req.UnmarshalProtobuf(data); err != nil {
				panic(err)
			}
			spanCount := 0
			for j := range req.ResourceSpans {
				for k := range req.ResourceSpans[j].ScopeSpans {
					spanCount += len(req.ResourceSpans[j].ScopeSpans[k].Spans)
					for l := range req.ResourceSpans[j].ScopeSpans[k].Spans {
						sp := req.ResourceSpans[j].ScopeSpans[k].Spans[l]
						once.Do(func() {
							timeOffset = uint64(time.Now().UnixNano()) - sp.StartTimeUnixNano
						})
						// replace TraceID
						if tid, ok := traceIDMap[sp.TraceID]; ok {
							sp.TraceID = tid
						} else {
							// generate one
							h := md5.New()
							h.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
							traceID := hex.EncodeToString(h.Sum(nil))
							traceIDMap[sp.TraceID] = traceID
							sp.TraceID = traceID
						}
						sp.StartTimeUnixNano = sp.StartTimeUnixNano + timeOffset
						sp.EndTimeUnixNano = sp.EndTimeUnixNano + timeOffset + uint64(rand.Int63n(100000000))
					}
				}
			}
			limiter.WaitN(context.TODO(), spanCount)
			_, err := http.Post(*addr, "application/x-protobuf", bytes.NewReader(req.MarshalProtobuf(nil)))
			if err != nil {
				logger.Errorf("trace export error: %s", err)
			}
		}
	}
}
