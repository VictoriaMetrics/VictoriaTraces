package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
)

func TestHandleSamplingRequestData(t *testing.T) {
	f := func(st []SamplingTrace, expectTidList [][16]byte) {
		data := unsampledRequestBuilder(st)

		unitTestDecisionVerifier := func(tidList [][16]byte) {
			if len(tidList) == 0 && len(expectTidList) == 0 {
				return
			}

			sort.Slice(tidList, func(i, j int) bool {
				return bytes.Compare(tidList[i][:], tidList[j][:]) < 0
			})
			sort.Slice(expectTidList, func(i, j int) bool {
				return bytes.Compare(tidList[i][:], tidList[j][:]) < 0
			})

			logger.Infof("expected traceID list = %v, got %v", expectTidList, tidList)
		}

		startBufCleaner(unitTestDecisionVerifier, time.Second)
		if err := handleSamplingRequestData(data); err != nil {
			t.Fatalf("handleSamplingRequestData() error = %v", err)
		}

		time.Sleep(4 * time.Second)
	}

	notSampledTid := genFakeTraceID()
	notSampledTraceList := []SamplingTrace{
		{
			TraceID:    notSampledTid,
			EndTime:    uint64(time.Now().UnixNano()),
			StartTime:  uint64(time.Now().Add(-5050 * time.Millisecond).UnixNano()),
			StatusCode: 0,
		},
	}

	sampledTid := [16]byte{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	sampledTraceList := []SamplingTrace{
		{
			TraceID:    sampledTid,
			EndTime:    uint64(time.Now().UnixNano()),
			StartTime:  uint64(time.Now().Add(-50 * time.Millisecond).UnixNano()),
			StatusCode: 0,
		},
		{
			TraceID:    sampledTid,
			EndTime:    uint64(time.Now().Add(-1500 * time.Millisecond).UnixNano()),
			StartTime:  uint64(time.Now().Add(-1600 * time.Millisecond).UnixNano()),
			StatusCode: 2,
		},
	}

	f(append(notSampledTraceList, sampledTraceList...), [][16]byte{{0}})
}

func TestUintConv(t *testing.T) {
	int64Time := time.Now().UnixNano()
	fmt.Println(int64Time)
	fmt.Println(uint64(int64Time))
}

func unsampledRequestBuilder(st []SamplingTrace) []byte {
	var result []byte

	for _, t := range st {
		result = append(result, decisionNotSampled)
		result = append(result, t.TraceID[:]...)

		result = binary.BigEndian.AppendUint64(result, t.StartTime)
		result = binary.BigEndian.AppendUint64(result, t.EndTime)
		result = append(result, byte(int8(t.StatusCode)))
	}
	return result
}

func sampledRequestBuilder(st []SamplingTrace) []byte {
	var result []byte

	for _, t := range st {
		result = append(result, decisionSampled)
		result = append(result, t.TraceID[:]...)
	}
	return result
}

func TestGenFakeTraceID(t *testing.T) {
	ftid := genFakeTraceID()
	fmt.Println(string(ftid[:]))
}

func genFakeTraceID() [16]byte {
	tid := [16]byte{}
	tidSlice := make([]byte, 0, 16)
	tidSlice = binary.BigEndian.AppendUint64(tidSlice, uint64(time.Now().UnixNano()))
	copy(tid[:], tidSlice)
	return tid
}
