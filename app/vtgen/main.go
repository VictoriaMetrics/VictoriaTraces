package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/encoding/zstd"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
	"github.com/VictoriaMetrics/metrics"
	"golang.org/x/time/rate"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var (
	httpListenAddrs     = flag.String("httpListenAddr", "0.0.0.0:8080", "http listen address for pprof and metrics.")
	spanRate            = flag.Int("rate", 10000, "spans per second.")
	addrs               = flag.String("addrs", "", `otlp trace export endpoints, split by ",".`)
	authHeaders         = flag.String("authorizations", "", `authorization headers for each -addrs, split by ",".`)
	worker              = flag.Int("worker", 4, "number of workers.")
	logNTraceIDEvery10K = flag.Int("logEvery10k", 2, "how many trace id should be logged for every 10000 traces by each worker.")
)

func main() {
	// parse and validate cli flags.
	addrList, authHeaderList := initFlags()

	// load test data from file.
	reqBodyList := loadTestData()

	// rate limit
	limiter := rate.NewLimiter(rate.Limit(*spanRate), *spanRate)

	// create metrics and pprof HTTP server
	requestHistogramList := make([]*metrics.Histogram, len(addrList))
	errCountList := make([]*metrics.Counter, len(addrList))
	for i, addr := range addrList {
		requestHistogramList[i] = metrics.NewHistogram(`vt_gen_http_request_duration_seconds{path="` + addr + `"}`)
		errCountList[i] = metrics.NewCounter(`vt_gen_http_request_error_count{path="` + addr + `"}`)
	}
	http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, true)
	})
	go func() {
		if err := http.ListenAndServe(*httpListenAddrs, nil); err != nil {
			logger.Fatalf("failed to start HTTP server: %s", err)
		}
	}()

	for i := 0; i < *worker; i++ {
		go func() {
			for {
				// The traceIDMap recorded old traceID->new traceID.
				// Spans with same old traceID should be replaced with same new traceID.
				traceIDMap := make(map[string]string)

				// The timeOffset is the time offset of span timestamp and current timestamp.
				// All spans' timestamp should be increased by this offset.
				// This value should be initialized only once when iterating through the first span.
				once := sync.Once{}
				timeOffset := uint64(0)

				// update the traceID and start_/end_timestamp of each span.
				for i := range reqBodyList {
					// the span count of this request. used for rate limiting.
					spanCount := 0

					// unmarshal binary request body to otelpb.ExportTraceServiceRequest
					data := reqBodyList[i]
					var req otelpb.ExportTraceServiceRequest
					if err := req.UnmarshalProtobuf(data); err != nil {
						panic(err)
					}

					// iterate all spans
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
									// old traceID already seen. use the cached one.
									sp.TraceID = tid
								} else {
									// generate a new traceID by md5(timestamp) and put it into cache.
									traceID := generateTraceID()
									oldTraceID := sp.TraceID
									sp.TraceID = traceID
									traceIDMap[oldTraceID] = traceID

									// log traceID for query test if needed.
									if rand.Intn(10000) < *logNTraceIDEvery10K {
										logger.Infof(traceID)
									}
								}

								// adjust the timestamp of the span.
								sp.StartTimeUnixNano = sp.StartTimeUnixNano + timeOffset
								sp.EndTimeUnixNano = sp.EndTimeUnixNano + timeOffset + uint64(rand.Int63n(100000000))
							}
						}
					}

					// rate limit
					_ = limiter.WaitN(context.TODO(), spanCount)

					// send request to each address.
					for addrIdx, addr := range addrList {
						// prepare request.
						httpReq, err := http.NewRequest("POST", addr, bytes.NewReader(req.MarshalProtobuf(nil)))
						if err != nil {
							logger.Errorf("cannot create http request for addr %q: %s", addr, err)
							continue
						}
						if *authHeaders != "" {
							httpReq.Header.Add("authorization", authHeaderList[addrIdx])
						}
						httpReq.Header.Add("content-type", "application/x-protobuf")

						// do request and record metrics.
						startTime := time.Now()
						res, err := http.DefaultClient.Do(httpReq)
						if err != nil {
							logger.Errorf("trace export error: %s", err)
							errCountList[addrIdx].Add(1)
						}
						if res != nil {
							res.Body.Close()
						}
						requestHistogramList[addrIdx].Update(time.Since(startTime).Seconds())
					}
				}
			}
		}()
	}
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
}

func initFlags() ([]string, []string) {
	flag.Parse()
	addrList := strings.Split(*addrs, ",")
	for _, addr := range addrList {
		if _, err := url.ParseRequestURI(addr); err != nil {
			panic(fmt.Sprintf("invalid otlp trace export endpoint %s: %v", addr, err))
		}
	}
	authHeaderList := strings.Split(*authHeaders, ",")
	if *authHeaders != "" && len(addrList) != len(authHeaderList) {
		panic("len(addrList) != len(authHeaderList)")
	}
	return addrList, authHeaderList
}

func loadTestData() [][]byte {
	var bodyList [][]byte

	// read compressed binary data
	data, err := os.ReadFile("./app/vtgen/testdata/testdata.bin")
	if err != nil {
		panic(fmt.Sprintf("cannot read file %v", err))
	}

	// decompress
	var uncompressed []byte
	uncompressed, err = zstd.Decompress(uncompressed, data)
	if err != nil {
		panic(fmt.Sprintf("cannot decompress %v", err))
	}

	// unmarshal binary data to the slice of request body ([]byte)
	gobDec := gob.NewDecoder(bytes.NewReader(uncompressed))
	if err = gobDec.Decode(&bodyList); err != nil {
		panic(fmt.Sprintf("cannot decode %v", err))
	}

	return bodyList
}

func generateTraceID() string {
	h := md5.New()
	h.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	return hex.EncodeToString(h.Sum(nil))
}

// readWrite Does the following:
// 1. read request body binary files like `1.bin`, `2.bin` and puts them into `BodyList`.
// 2. encode and compress the `BodyList` into `[]byte`.
// 3. write the `[]byte` result to `./app/vtgen/testdata/testdata.bin`.
//
// You have to prepare the request body binary in advance.
//func readWrite() {
//	var bodyList [][]byte
//	for i := 0; i <= 100; i++ {
//		dat, err := os.ReadFile(fmt.Sprintf("%d.bin", i))
//		if err != nil {
//			panic(fmt.Sprintf("cannot read file %d: %v", i, err))
//		}
//		bodyList = append(bodyList, dat)
//	}
//
//	var buf bytes.Buffer
//	gobEnc := gob.NewEncoder(&buf)
//	if err := gobEnc.Encode(bodyList); err != nil {
//		panic(err)
//	}
//	var compressed []byte
//	compressed = zstd.CompressLevel(compressed, buf.Bytes(), 3)
//	os.WriteFile("./app/vtgen/testdata/testdata.bin", compressed, 0666)
//}
