package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
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
	grpcMode            = flag.Bool("grpcMode", false, "send data in otlp grpc instead of otlp http.")
)

var (
	http2Client          *http.Client
	requestHistogramList []*metrics.Histogram
	errCountList         []*metrics.Counter
)

func main() {
	// parse and validate cli flags, init metrics
	addrList, authHeaderList := initFlagsAndMetrics()

	// load test data from file.
	reqBodyList := loadTestData()

	// rate limit
	limiter := rate.NewLimiter(rate.Limit(*spanRate), *spanRate)

	// create metrics and pprof HTTP server
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
				doHTPPRequest(reqBodyList, limiter, addrList, authHeaderList)
			}
		}()
	}
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
}

func initFlagsAndMetrics() ([]string, []string) {
	// init flags
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

	// init metrics
	requestHistogramList = make([]*metrics.Histogram, len(addrList))
	errCountList = make([]*metrics.Counter, len(addrList))
	for i, addr := range addrList {
		requestHistogramList[i] = metrics.NewHistogram(`vt_gen_request_duration_seconds{addr="` + addr + `"}`)
		errCountList[i] = metrics.NewCounter(`vt_gen_request_error_count{addr="` + addr + `"}`)
	}

	// init HTTP2 client
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	http2Client = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			Protocols:         &protocols,
		},
	}

	// return
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

func doHTPPRequest(reqBodyList [][]byte, limiter *rate.Limiter, addrList, authHeaderList []string) {
	// The traceIDMap recorded old traceID->new traceID.
	// Spans with same old traceID should be replaced with same new traceID.
	traceIDMap := make(map[string]string)
	spanIDMap := make(map[string]string)

	// The timeOffset is the time offset of span timestamp and current timestamp.
	// All spans' timestamp should be increased by this offset.
	// This value should be initialized only once when iterating through the first span.
	initTimeOnce := sync.Once{}
	timeOffset := uint64(0)

	// update the traceID and start_/end_timestamp of each span.
	for idx := range reqBodyList {
		spanCount := 0
		var req otelpb.ExportTraceServiceRequest

		// unmarshal binary request body to otelpb.ExportTraceServiceRequest
		data := reqBodyList[idx]
		if err := req.UnmarshalProtobuf(data); err != nil {
			panic(err)
		}

		// iterate all spans
		for j := range req.ResourceSpans {
			for k := range req.ResourceSpans[j].ScopeSpans {
				spanCount += len(req.ResourceSpans[j].ScopeSpans[k].Spans)

				for l := range req.ResourceSpans[j].ScopeSpans[k].Spans {
					sp := req.ResourceSpans[j].ScopeSpans[k].Spans[l]

					initTimeOnce.Do(func() {
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

					// replace SpanID
					if sid, ok := spanIDMap[sp.SpanID]; ok {
						sp.SpanID = sid
					} else {
						spanID := generateSpanID()
						oldSpanID := sp.SpanID
						sp.SpanID = spanID
						spanIDMap[oldSpanID] = spanID
					}

					// replace parentSpanID
					if sid, ok := spanIDMap[sp.ParentSpanID]; ok {
						sp.ParentSpanID = sid
					} else {
						parentSpanID := generateSpanID()
						oldParentSpanID := sp.ParentSpanID
						sp.ParentSpanID = parentSpanID
						spanIDMap[oldParentSpanID] = parentSpanID
					}

					// adjust the timestamp of the span.
					sp.StartTimeUnixNano = sp.StartTimeUnixNano + timeOffset
					sp.EndTimeUnixNano = sp.EndTimeUnixNano + timeOffset + uint64(rand.Int63n(100000000))
				}
			}
		}

		// rate limit
		_ = limiter.WaitN(context.TODO(), spanCount)

		// for OTLPHTTP, the request body is the marshaled ExportTraceServiceRequest
		reqBytes := req.MarshalProtobuf(nil)
		if *grpcMode {
			// for OTLP in gRPC, it requires extra 5 bytes as flag, and then the marshaled ExportTraceServiceRequest as body.
			flagBytes := make([]byte, 5)
			binary.BigEndian.PutUint32(flagBytes[1:5], uint32(len(reqBytes)))
			// this is not efficient, but easy to understand and ok for test tool.
			reqBytes = append(flagBytes, reqBytes...)
		}

		// send request to each address.
		for addrIdx, addr := range addrList {
			var (
				httpReq    *http.Request
				err        error
				httpClient = http.DefaultClient
			)

			// prepare request.
			if *grpcMode {
				httpReq, err = http.NewRequest("POST", addr+"/opentelemetry.proto.collector.trace.v1.TraceService/Export", bytes.NewReader(reqBytes))
				if err != nil {
					logger.Errorf("cannot create http request for addr %q: %s", addr, err)
					continue
				}
				httpReq.Header.Add("Content-Type", "application/grpc")
				httpReq.Header.Add("Grpc-Accept-Encoding", "snappy,zstd,gzip,zstdarrow1,zstdarrow2,zstdarrow3,zstdarrow4,zstdarrow5,zstdarrow6,zstdarrow7,zstdarrow8,zstdarrow9,zstdarrow10")
				httpReq.Header.Add("Grpc-Timeout", "5000000u")
				httpReq.Header.Add("Te", "trailers")
				httpReq.Header.Add("User-Agent", "vtgen")
				httpClient = http2Client
			} else {
				httpReq, err = http.NewRequest("POST", addr, bytes.NewReader(reqBytes))
				if err != nil {
					logger.Errorf("cannot create http request for addr %q: %s", addr, err)
					continue
				}
				httpReq.Header.Add("content-type", "application/x-protobuf")
			}

			if *authHeaders != "" {
				httpReq.Header.Add("authorization", authHeaderList[addrIdx])
			}

			// do request and record metrics.
			startTime := time.Now()
			res, err := httpClient.Do(httpReq)
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

var traceIDMutex sync.Mutex

func generateTraceID() string {
	traceIDMutex.Lock()
	defer traceIDMutex.Unlock()

	h := md5.New()
	h.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	return hex.EncodeToString(h.Sum(nil))
}

var spanIDMutex sync.Mutex

func generateSpanID() string {
	spanIDMutex.Lock()
	defer spanIDMutex.Unlock()
	h := md5.New()
	h.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// readWrite Does the following:
// 1. read request body binary files like `1.bin`, `2.bin` and puts them into `BodyList`.
// 2. encode and compress the `BodyList` into `[]byte`.
// 3. write the `[]byte` result to `./app/vtgen/testdata/testdata.bin`.
//
// You have to prepare the request body binary in advance.
//func readWrite() {
//	var bodyList [][]byte
//	for i := 0; i <= 99; i++ {
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
//	os.WriteFile("./app/vtgen/testdata/testdata_grpc.bin", compressed, 0666)
//}
