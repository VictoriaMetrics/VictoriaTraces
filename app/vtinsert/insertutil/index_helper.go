package insertutil

import (
	"flag"
	"strconv"
	"sync"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/cgroup"
	"github.com/cespare/xxhash/v2"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

var (
	indexFlushInterval = flag.Duration("insert.indexFlushInterval", 20*time.Second, "Amount of time after which the index of a trace is flushed. VictoriaTraces creates an index for each trace ID based on its start and end times."+
		"Each trace ID must wait in the queue for -insert.indexFlushInterval, continuously updating its start and end times before being flushed into the index.")
)

type indexEntry struct {
	tenantID      logstorage.TenantID
	startTimeNano int64
	endTimeNano   int64
}

type indexWorker struct {
	// traceIDIndexMapCur and traceIDIndexMapPrev holds the index data *indexEntry for each traceID, before they could be persisted.
	// it mainly tracks the start time and end time of a trace, which is keep changing before they're persisted.
	//
	// - The cur map can accept new traceID and indexEntry.
	// - The prev map only serves for fast lookup of existing indexEntry.
	mu                  sync.Mutex
	traceIDIndexMapCur  map[[32]byte]indexEntry
	traceIDIndexMapPrev map[[32]byte]indexEntry

	// logMessageProcessorMap holds lmp for different tenants.
	logMessageProcessorMap map[logstorage.TenantID]LogMessageProcessor
}

var (
	workers []*indexWorker

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
		mustPushIndex(tenantID, traceID, startTime, endTime)
	}
	return true
}

// mustPushIndex compose an (or update an existing) indexEntry with tenantID, startTime and endTime for a trace,
// and put it to the traceIDIndex map.
// The indexEntry should indicate the real min(startTime) and max(endTime) of a trace, and be flushed to disk later.
func mustPushIndex(tenantID logstorage.TenantID, traceID string, startTime, endTime int64) {
	tb := [32]byte{}
	copy(tb[:], traceID)

	// todo: need a better hashing here
	worker := workers[int(tb[7]+tb[15]+tb[23]+tb[31])%len(workers)]

	worker.mu.Lock()
	defer worker.mu.Unlock()

	// find the (potential) existing indexEntry in both map.
	// if found, update the startTime and endTime for this trace.
	idxEntry, ok := worker.traceIDIndexMapCur[tb]
	if ok {
		idxEntry.startTimeNano = min(startTime, idxEntry.startTimeNano)
		idxEntry.endTimeNano = max(endTime, idxEntry.endTimeNano)
		worker.traceIDIndexMapCur[tb] = idxEntry
		return
	}

	idxEntry, ok = worker.traceIDIndexMapPrev[tb]
	if ok {
		idxEntry.startTimeNano = min(startTime, idxEntry.startTimeNano)
		idxEntry.endTimeNano = max(endTime, idxEntry.endTimeNano)
		worker.traceIDIndexMapPrev[tb] = idxEntry
		return
	}

	// this trace is new, compose an indexEntry and put it to the current map.
	idxEntry = indexEntry{}
	idxEntry.tenantID = tenantID
	idxEntry.startTimeNano = startTime
	idxEntry.endTimeNano = endTime

	worker.traceIDIndexMapCur[tb] = idxEntry
}

// MustStartIndexWorker starts a single goroutine indexWorker that reads from traceIDCh and write the index entry to storage.
func MustStartIndexWorker() {
	n := cgroup.AvailableCPUs()
	workers = make([]*indexWorker, n)
	for i := 0; i < n; i++ {
		workers[i] = &indexWorker{
			mu:                     sync.Mutex{},
			traceIDIndexMapCur:     make(map[[32]byte]indexEntry),
			traceIDIndexMapPrev:    make(map[[32]byte]indexEntry),
			logMessageProcessorMap: make(map[logstorage.TenantID]LogMessageProcessor),
		}

		indexWorkerWg.Add(1)
		go workers[i].run()
	}
}

func (w *indexWorker) run() {
	defer indexWorkerWg.Done()

	ticker := time.NewTicker(*indexFlushInterval / 2)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			// persist all the index in the queue,
			// even though they're still fresh (haven't waited for *indexFlushInterval).
			w.mu.Lock()
			for k, v := range w.traceIDIndexMapPrev {
				w.flushIndexInMap(k, v)
			}
			for k, v := range w.traceIDIndexMapCur {
				w.flushIndexInMap(k, v)
			}
			for _, lmp := range w.logMessageProcessorMap {
				lmp.MustClose()
			}
			w.mu.Unlock()

			return
		case <-ticker.C:
			// flush the data in prev map
			w.mu.Lock()

			for k, v := range w.traceIDIndexMapPrev {
				w.flushIndexInMap(k, v)
			}
			// swap the empty prev map as the new current map.
			n := len(w.traceIDIndexMapPrev)

			// drop the previous map and create a new one
			w.traceIDIndexMapPrev = make(map[[32]byte]indexEntry, n)

			// swap the previous map and current map
			w.traceIDIndexMapCur, w.traceIDIndexMapPrev = w.traceIDIndexMapPrev, w.traceIDIndexMapCur

			w.mu.Unlock()
		}
	}
}

// flushIndexInMap flush the in-memory index to log streams.
func (w *indexWorker) flushIndexInMap(tb [32]byte, idxEntry indexEntry) bool {
	lmp, ok := w.logMessageProcessorMap[idxEntry.tenantID]
	if !ok {
		// init the lmp for the current tenant
		cp := CommonParams{
			TenantID:   idxEntry.tenantID,
			TimeFields: []string{"_time"},
		}
		lmp = cp.NewLogMessageProcessor("internalinsert_index", true)

		// only current goroutine can read/write this map, so mutex is not needed.
		// consider adding a mutex if index indexWorker is scaled to multi-goroutines.
		w.logMessageProcessorMap[idxEntry.tenantID] = lmp
	}

	startTimestamp := idxEntry.startTimeNano
	endTimestamp := idxEntry.endTimeNano
	lmp.AddRow(startTimestamp,
		// fields
		[]logstorage.Field{
			{Name: otelpb.TraceIDIndexStreamName, Value: strconv.FormatUint(xxhash.Sum64(tb[:])%otelpb.TraceIDIndexPartitionCount, 10)},
			{Name: "_msg", Value: "-"},
			{Name: otelpb.TraceIDIndexFieldName, Value: string(tb[:])},
			{Name: otelpb.TraceIDIndexStartTimeFieldName, Value: strconv.FormatInt(startTimestamp, 10)},
			{Name: otelpb.TraceIDIndexEndTimeFieldName, Value: strconv.FormatInt(endTimestamp, 10)},
			{Name: otelpb.TraceIDIndexDuration, Value: strconv.FormatInt(endTimestamp-startTimestamp, 10)},
		},
		1,
	)
	return true
}

func MustStopIndexWorker() {
	close(stopCh)

	// wait until all the index workers exit
	indexWorkerWg.Wait()
}
