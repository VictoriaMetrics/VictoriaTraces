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
	"golang.org/x/time/rate"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var BodyList [][]byte

func main() {
	spanRate := flag.Int("rate", 10000, "spans per second.")
	addrs := flag.String("addrs", "", "otlp trace export endpoint.")
	authHeaders := flag.String("authorizations", "", "authorization header.")
	worker := flag.Int("worker", 4, "number of workers.")
	logNTraceIDEvery10K := flag.Int("logEvery10k", 2, "how many trace id should be logged for every 10000 traces for each worker.")

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

	data, err := os.ReadFile("./app/vtgen/testdata/testdata.bin")
	if err != nil {
		panic(fmt.Sprintf("cannot read file %v", err))
	}
	var uncompressed []byte
	uncompressed, err = zstd.Decompress(uncompressed, data)
	if err != nil {
		panic(fmt.Sprintf("cannot decompress %v", err))
	}

	gobDec := gob.NewDecoder(bytes.NewReader(uncompressed))
	if err = gobDec.Decode(&BodyList); err != nil {
		panic(fmt.Sprintf("cannot decode %v", err))
	}

	go http.ListenAndServe("0.0.0.0:8080", nil)

	limiter := rate.NewLimiter(rate.Limit(*spanRate), *spanRate)
	wg := sync.WaitGroup{}
	for i := 0; i < *worker; i++ {
		wg.Add(1)
		go func() {
			for {
				traceIDMap := make(map[string]string)
				once := sync.Once{}
				timeOffset := uint64(0)
				for i := range BodyList {
					data := BodyList[i]
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
									if rand.Intn(10000) < *logNTraceIDEvery10K {
										logger.Infof(traceID)
									}
								}
								sp.StartTimeUnixNano = sp.StartTimeUnixNano + timeOffset
								sp.EndTimeUnixNano = sp.EndTimeUnixNano + timeOffset + uint64(rand.Int63n(100000000))
							}
						}
					}
					limiter.WaitN(context.TODO(), spanCount)
					for addrIdx, addr := range addrList {
						httpReq, err := http.NewRequest("POST", addr, bytes.NewReader(req.MarshalProtobuf(nil)))
						if *authHeaders != "" {
							httpReq.Header.Add("authorization", authHeaderList[addrIdx])
						}
						httpReq.Header.Add("content-type", "application/x-protobuf")
						res, err := http.DefaultClient.Do(httpReq)
						if err != nil {
							logger.Errorf("trace export error: %s", err)
						}
						res.Body.Close()
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

// readWrite Does the following:
// 1. read request body binary files like `1.bin`, `2.bin` and puts them into `BodyList`.
// 2. encode and compress the `BodyList` into `[]byte`.
// 3. write the `[]byte` result to `./app/vtgen/testdata/testdata.bin`.
//
// You have to prepare the request body binary in advance.
func readWrite() {
	for i := 0; i <= 100; i++ {
		dat, err := os.ReadFile(fmt.Sprintf("%d.bin", i))
		if err != nil {
			panic(fmt.Sprintf("cannot read file %d: %v", i, err))
		}
		BodyList = append(BodyList, dat)
	}

	var buf bytes.Buffer
	gobEnc := gob.NewEncoder(&buf)
	if err := gobEnc.Encode(BodyList); err != nil {
		panic(err)
	}
	var compressed []byte
	compressed = zstd.CompressLevel(compressed, buf.Bytes(), 3)
	os.WriteFile("./app/vtgen/testdata/testdata.bin", compressed, 0666)
}
