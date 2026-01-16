package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
	"github.com/VictoriaMetrics/metrics"
	"golang.org/x/time/rate"
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
	limiter := rate.NewLimiter(rate.Limit(426), 426)

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
	for i := 1; i < 103; i++ {
		data, err := os.ReadFile(fmt.Sprintf(`./app/vlgen/testdata/%d.json`, i))
		if err != nil {
			panic(fmt.Sprintf("cannot read file %v", err))
		}
		data = []byte(strings.TrimSuffix(string(data), "\n"))
		bodyList = append(bodyList, data)
	}
	return bodyList
}

func doHTPPRequest(reqBodyList [][]byte, limiter *rate.Limiter, addrList, authHeaderList []string) {
	// update the traceID and start_/end_timestamp of each span.
	for idx := range reqBodyList {
		// unmarshal binary request body to otelpb.ExportTraceServiceRequest
		data := reqBodyList[idx]

		// rate limit
		_ = limiter.WaitN(context.TODO(), 1)

		// for OTLPHTTP, the request body is the marshaled ExportTraceServiceRequest
		reqBytes := data
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
