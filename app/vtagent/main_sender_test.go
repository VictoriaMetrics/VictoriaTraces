package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
)

// this is a manual sender to send data to the sampling server.
func TestSendSamplingRequest(t *testing.T) {
	sampledTid1 := [16]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 3, 9}
	sampledTraceList1 := []SamplingTrace{
		{
			TraceID:    sampledTid1,
			EndTime:    uint64(time.Now().UnixNano()),
			StartTime:  uint64(time.Now().Add(-50 * time.Millisecond).UnixNano()),
			StatusCode: 0,
		},
	}
	b := unsampledRequestBuilder(sampledTraceList1)

	sampledTid2 := [16]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 1, 9}
	sampledTraceList2 := []SamplingTrace{
		{
			TraceID: sampledTid2,
		},
	}
	b = append(b, sampledRequestBuilder(sampledTraceList2)...)

	resp, err := http.Post("http://127.0.0.1:10430/insert/sampling", "application/octet-stream", bytes.NewReader(b))
	if err != nil {
		logger.Errorf("post failed: %s", err)
	}
	defer resp.Body.Close()

	readAll, _ := io.ReadAll(resp.Body)
	logger.Infof("post succeeded: %s, %s", resp.Status, string(readAll))
}
