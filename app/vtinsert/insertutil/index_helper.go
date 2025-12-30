package insertutil

import (
	"flag"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/cespare/xxhash/v2"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var (
	traceMaxDuration = flag.Duration("insert.indexFlushInterval", 30*time.Second, "Amount of time after which the index of a trace is flushed. VictoriaTraces creates an index for each trace ID based on its start and end times."+
		"Each trace ID must wait in the queue for -insert.indexFlushInterval, continuously updating its start and end times before being flushed into the index.")
)

const int64Max = int64(1<<63 - 1)

type indexEntry struct {
	tenantID      logstorage.TenantID
	startTimeNano atomic.Int64
	endTimeNano   atomic.Int64
}

var (
	// traceIDIndexMapCur and traceIDIndexMapPrev holds the index data *indexEntry for each traceID, before they could be persisted.
	// it mainly tracks the start time and end time of a trace, which could be edited before they're persisted.
	//
	// - The cur map can accept new traceID and *indexEntry.
	// - The prev map only serves for fast lookup of existing *indexEntry.
	traceIDIndexMapCur  = &sync.Map{}
	traceIDIndexMapPrev = &sync.Map{}

	// logMessageProcessorMap holds lmp for different tenants.
	logMessageProcessorMap = make(map[logstorage.TenantID]LogMessageProcessor)

	// indexWorkerWg is the WaitGroup for IndexWorker. indexWorkerWg.Wait() should be used during shutdown.
	indexWorkerWg = sync.WaitGroup{}
	stopCh        = make(chan struct{})
)

// pushIndexToQueue organize index data (from LogMessageProcessor interface or InsertRowProcessor interface)
// and push it to the queue.
func pushIndexToQueue(tenantID logstorage.TenantID, traceID string, startTime, endTime int64) bool {
	select {
	case <-stopCh:
		// during stop, no data should be pushed to the queue anymore.
		return false
	default:
		index, ok := traceIDIndexMapCur.Load(traceID)
		if ok {
			idxEntry := index.(*indexEntry)
			for {
				st := idxEntry.startTimeNano.Load()
				if st < startTime {
					break
				}
				if idxEntry.startTimeNano.CompareAndSwap(st, startTime) {
					break
				}
			}
			for {
				et := idxEntry.endTimeNano.Load()
				if et > endTime {
					break
				}
				if idxEntry.endTimeNano.CompareAndSwap(et, endTime) {
					break
				}
			}
			return true
		}

		index, ok = traceIDIndexMapPrev.Load(traceID)
		if ok {
			idxEntry := index.(*indexEntry)
			for {
				st := idxEntry.startTimeNano.Load()
				if st < startTime {
					break
				}
				if idxEntry.startTimeNano.CompareAndSwap(st, startTime) {
					break
				}
			}
			for {
				et := idxEntry.endTimeNano.Load()
				if et > endTime {
					break
				}
				if idxEntry.endTimeNano.CompareAndSwap(et, endTime) {
					break
				}
			}
			return true
		}

		idxEntry := GetIndexEntry()
		idxEntry.tenantID = tenantID
		idxEntry.startTimeNano.Store(startTime)
		idxEntry.endTimeNano.Store(endTime)

		traceIDIndexMapCur.Store(traceID, idxEntry)
	}

	return true
}

// MustStartIndexWorker starts a single goroutine worker that reads from traceIDCh and write the index entry to storage.
func MustStartIndexWorker() {
	indexWorkerWg.Add(1)
	go func() {
		defer indexWorkerWg.Done()

		ticker := time.NewTicker(*traceMaxDuration / 2)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				// persist all the index in the queue,
				// even though they're still fresh (haven't waited for *traceMaxDuration).
				traceIDIndexMapPrev.Range(flushIndexInMap)
				traceIDIndexMapCur.Range(flushIndexInMap)

				return
			case <-ticker.C:
				// flush the data in prev map
				traceIDIndexMapPrev.Range(func(k, v any) bool {
					return flushIndexInMap(k, v)
				})
				// swap the empty prev map as the new current map.
				traceIDIndexMapPrev.Clear()
				traceIDIndexMapCur, traceIDIndexMapPrev = traceIDIndexMapPrev, traceIDIndexMapCur
			}
		}
	}()
}

// flushIndexInMap flush the in-memory index to log streams.
func flushIndexInMap(traceID, index any) bool {
	idxEntry := index.(*indexEntry)
	defer PutIndexEntry(idxEntry)

	lmp, ok := logMessageProcessorMap[idxEntry.tenantID]
	if !ok {
		// init the lmp for the current tenant
		cp := CommonParams{
			TenantID:   idxEntry.tenantID,
			TimeFields: []string{"_time"},
		}
		lmp = cp.NewLogMessageProcessor("internalinsert_index", true)

		// only current goroutine can read/write this map, so mutex is not needed.
		// consider adding a mutex if index worker is scaled to multi-goroutines.
		logMessageProcessorMap[idxEntry.tenantID] = lmp
	}

	startTimestamp := idxEntry.startTimeNano.Load()
	endTimestamp := idxEntry.startTimeNano.Load()
	lmp.AddRow(startTimestamp,
		// fields
		[]logstorage.Field{
			{Name: "_msg", Value: "-"},
			{Name: otelpb.TraceIDIndexFieldName, Value: traceID.(string)},
			{Name: otelpb.TraceIDIndexStartTimeFieldName, Value: strconv.FormatInt(startTimestamp, 10)},
			{Name: otelpb.TraceIDIndexEndTimeFieldName, Value: strconv.FormatInt(endTimestamp, 10)},
		},
		// stream fields
		[]logstorage.Field{
			{Name: otelpb.TraceIDIndexStreamName, Value: strconv.FormatUint(xxhash.Sum64String(traceID.(string))%otelpb.TraceIDIndexPartitionCount, 10)},
		},
	)
	return true
}

func MustStopIndexWorker() {
	close(stopCh)

	// wait until all the index workers exit
	indexWorkerWg.Wait()

	for _, lmp := range logMessageProcessorMap {
		lmp.MustClose()
	}
}

var indexEntryPool = &sync.Pool{
	New: func() any {
		return &indexEntry{}
	},
}

// GetIndexEntry return a *indexEntry from the pool.
func GetIndexEntry() *indexEntry {
	return indexEntryPool.Get().(*indexEntry)
}

// PutIndexEntry returns a *indexEntry back to the pool.
func PutIndexEntry(x *indexEntry) {
	// reset all the fields
	x.tenantID.Reset()
	x.startTimeNano.Store(int64Max)
	x.endTimeNano.Store(0)

	indexEntryPool.Put(x)
}
