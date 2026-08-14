package logstorage

import (
	"fmt"
	"sync/atomic"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/bytesutil"
)

// QueryStats contains various query execution stats.
type QueryStats struct {
	// BytesReadColumnsHeaders is the total number of columns header bytes read from disk during the search.
	BytesReadColumnsHeaders uint64

	// BytesReadColumnsHeaderIndexes is the total number of columns header index bytes read from disk during the search.
	BytesReadColumnsHeaderIndexes uint64

	// BytesReadBloomFilters is the total number of bloom filter bytes read from disk during the search.
	BytesReadBloomFilters uint64

	// BytesReadValues is the total number of values bytes read from disk during the search.
	BytesReadValues uint64

	// BytesReadTimestamps is the total number of timestamps bytes read from disk during the search.
	BytesReadTimestamps uint64

	// BytesReadBlockHeaders is the total number of headers bytes read from disk during the search.
	BytesReadBlockHeaders uint64

	// BlocksProcessed is the number of data blocks processed during query execution.
	BlocksProcessed uint64

	// RowsProcessed is the number of log rows processed during query execution.
	RowsProcessed uint64

	// RowsFound is the number of rows found by the query.
	RowsFound uint64

	// ValuesRead is the number of log field values read during query exection.
	ValuesRead uint64

	// TimestampsRead is the number of timestamps read during query execution.
	TimestampsRead uint64

	// BytesProcessedUncompressedValues is the total number of uncompressed values bytes processed during the search.
	BytesProcessedUncompressedValues uint64

	// PartitionsTotal is the number of partitions present at the time-range selection boundary.
	PartitionsTotal uint64

	// PartitionsSelected is the number of partitions selected by the query time range.
	PartitionsSelected uint64

	// PartitionsTimeSkipped is the number of partitions skipped by the query time range.
	PartitionsTimeSkipped uint64

	// PartsTotal is the number of parts present at the time-range selection boundaries.
	PartsTotal uint64

	// PartsSelected is the number of parts selected by the query time range.
	PartsSelected uint64

	// PartsTimeSkipped is the number of parts skipped by the query time range.
	PartsTimeSkipped uint64

	// IndexBlockHeadersConsidered is the number of index block headers inspected by key traversal.
	IndexBlockHeadersConsidered uint64

	// IndexBlockHeadersRead is the number of considered index block headers whose block headers were read.
	IndexBlockHeadersRead uint64

	// IndexBlockHeadersTimeOrKeySkipped is the number of considered index block headers skipped before reading.
	IndexBlockHeadersTimeOrKeySkipped uint64

	// BlockHeadersDecoded is the number of block headers decoded from index blocks.
	BlockHeadersDecoded uint64

	// BlockHeadersTimeOrKeySkipped is the number of decoded block headers skipped before scheduling.
	BlockHeadersTimeOrKeySkipped uint64

	// BlocksScheduled is the number of block searches dispatched to workers.
	BlocksScheduled uint64

	// BlocksCancelledBeforeProcess is the number of dispatched block searches skipped at the worker cancellation gate.
	BlocksCancelledBeforeProcess uint64

	// RowsSkippedBeforeScheduling is the number of rows represented by decoded block headers skipped before scheduling.
	RowsSkippedBeforeScheduling uint64

	detailedProfilingEnabled uint32
}

// QueryStatsSnapshot is an immutable, JSON-friendly point-in-time copy of QueryStats.
type QueryStatsSnapshot struct {
	BytesReadTotal                    uint64 `json:"bytes_read_total"`
	BytesReadColumnsHeaders           uint64 `json:"bytes_read_columns_headers"`
	BytesReadColumnsHeaderIndexes     uint64 `json:"bytes_read_columns_header_indexes"`
	BytesReadBloomFilters             uint64 `json:"bytes_read_bloom_filters"`
	BytesReadValues                   uint64 `json:"bytes_read_values"`
	BytesReadTimestamps               uint64 `json:"bytes_read_timestamps"`
	BytesReadBlockHeaders             uint64 `json:"bytes_read_block_headers"`
	BlocksProcessed                   uint64 `json:"blocks_processed"`
	RowsProcessed                     uint64 `json:"rows_processed"`
	RowsFound                         uint64 `json:"rows_found"`
	ValuesRead                        uint64 `json:"values_read"`
	TimestampsRead                    uint64 `json:"timestamps_read"`
	BytesProcessedUncompressedValues  uint64 `json:"bytes_processed_uncompressed_values"`
	PartitionsTotal                   uint64 `json:"partitions_total"`
	PartitionsSelected                uint64 `json:"partitions_selected"`
	PartitionsTimeSkipped             uint64 `json:"partitions_time_skipped"`
	PartsTotal                        uint64 `json:"parts_total"`
	PartsSelected                     uint64 `json:"parts_selected"`
	PartsTimeSkipped                  uint64 `json:"parts_time_skipped"`
	IndexBlockHeadersConsidered       uint64 `json:"index_block_headers_considered"`
	IndexBlockHeadersRead             uint64 `json:"index_block_headers_read"`
	IndexBlockHeadersTimeOrKeySkipped uint64 `json:"index_block_headers_time_or_key_skipped"`
	BlockHeadersDecoded               uint64 `json:"block_headers_decoded"`
	BlockHeadersTimeOrKeySkipped      uint64 `json:"block_headers_time_or_key_skipped"`
	BlocksScheduled                   uint64 `json:"blocks_scheduled"`
	BlocksCancelledBeforeProcess      uint64 `json:"blocks_cancelled_before_process"`
	RowsSkippedBeforeScheduling       uint64 `json:"rows_skipped_before_scheduling"`
	DetailedProfilingEnabled          bool   `json:"detailed_profiling_enabled"`
}

// EnableDetailedProfiling enables collection of the opt-in data-layout counters.
// Once enabled, it remains enabled for the lifetime of qs.
func (qs *QueryStats) EnableDetailedProfiling() {
	if qs != nil {
		atomic.StoreUint32(&qs.detailedProfilingEnabled, 1)
	}
}

// DetailedProfilingEnabled reports whether opt-in data-layout counters are enabled.
func (qs *QueryStats) DetailedProfilingEnabled() bool {
	return qs != nil && atomic.LoadUint32(&qs.detailedProfilingEnabled) != 0
}

// Snapshot returns a concurrency-safe point-in-time copy of qs.
func (qs *QueryStats) Snapshot() QueryStatsSnapshot {
	if qs == nil {
		return QueryStatsSnapshot{}
	}
	snapshot := QueryStatsSnapshot{
		BytesReadColumnsHeaders:           atomic.LoadUint64(&qs.BytesReadColumnsHeaders),
		BytesReadColumnsHeaderIndexes:     atomic.LoadUint64(&qs.BytesReadColumnsHeaderIndexes),
		BytesReadBloomFilters:             atomic.LoadUint64(&qs.BytesReadBloomFilters),
		BytesReadValues:                   atomic.LoadUint64(&qs.BytesReadValues),
		BytesReadTimestamps:               atomic.LoadUint64(&qs.BytesReadTimestamps),
		BytesReadBlockHeaders:             atomic.LoadUint64(&qs.BytesReadBlockHeaders),
		BlocksProcessed:                   atomic.LoadUint64(&qs.BlocksProcessed),
		RowsProcessed:                     atomic.LoadUint64(&qs.RowsProcessed),
		RowsFound:                         atomic.LoadUint64(&qs.RowsFound),
		ValuesRead:                        atomic.LoadUint64(&qs.ValuesRead),
		TimestampsRead:                    atomic.LoadUint64(&qs.TimestampsRead),
		BytesProcessedUncompressedValues:  atomic.LoadUint64(&qs.BytesProcessedUncompressedValues),
		PartitionsTotal:                   atomic.LoadUint64(&qs.PartitionsTotal),
		PartitionsSelected:                atomic.LoadUint64(&qs.PartitionsSelected),
		PartitionsTimeSkipped:             atomic.LoadUint64(&qs.PartitionsTimeSkipped),
		PartsTotal:                        atomic.LoadUint64(&qs.PartsTotal),
		PartsSelected:                     atomic.LoadUint64(&qs.PartsSelected),
		PartsTimeSkipped:                  atomic.LoadUint64(&qs.PartsTimeSkipped),
		IndexBlockHeadersConsidered:       atomic.LoadUint64(&qs.IndexBlockHeadersConsidered),
		IndexBlockHeadersRead:             atomic.LoadUint64(&qs.IndexBlockHeadersRead),
		IndexBlockHeadersTimeOrKeySkipped: atomic.LoadUint64(&qs.IndexBlockHeadersTimeOrKeySkipped),
		BlockHeadersDecoded:               atomic.LoadUint64(&qs.BlockHeadersDecoded),
		BlockHeadersTimeOrKeySkipped:      atomic.LoadUint64(&qs.BlockHeadersTimeOrKeySkipped),
		BlocksScheduled:                   atomic.LoadUint64(&qs.BlocksScheduled),
		BlocksCancelledBeforeProcess:      atomic.LoadUint64(&qs.BlocksCancelledBeforeProcess),
		RowsSkippedBeforeScheduling:       atomic.LoadUint64(&qs.RowsSkippedBeforeScheduling),
		DetailedProfilingEnabled:          atomic.LoadUint32(&qs.detailedProfilingEnabled) != 0,
	}
	snapshot.BytesReadTotal = snapshot.BytesReadColumnsHeaders + snapshot.BytesReadColumnsHeaderIndexes + snapshot.BytesReadBloomFilters + snapshot.BytesReadValues + snapshot.BytesReadTimestamps + snapshot.BytesReadBlockHeaders
	return snapshot
}

// GetBytesReadTotal returns the total number of bytes read, which is tracked by qs.
func (qs *QueryStats) GetBytesReadTotal() uint64 {
	return qs.BytesReadColumnsHeaders + qs.BytesReadColumnsHeaderIndexes + qs.BytesReadBloomFilters + qs.BytesReadValues + qs.BytesReadTimestamps + qs.BytesReadBlockHeaders
}

// UpdateAtomic add src to qs in an atomic manner.
func (qs *QueryStats) UpdateAtomic(src *QueryStats) {
	atomic.AddUint64(&qs.BytesReadColumnsHeaders, src.BytesReadColumnsHeaders)
	atomic.AddUint64(&qs.BytesReadColumnsHeaderIndexes, src.BytesReadColumnsHeaderIndexes)
	atomic.AddUint64(&qs.BytesReadBloomFilters, src.BytesReadBloomFilters)
	atomic.AddUint64(&qs.BytesReadValues, src.BytesReadValues)
	atomic.AddUint64(&qs.BytesReadTimestamps, src.BytesReadTimestamps)
	atomic.AddUint64(&qs.BytesReadBlockHeaders, src.BytesReadBlockHeaders)

	atomic.AddUint64(&qs.BlocksProcessed, src.BlocksProcessed)
	atomic.AddUint64(&qs.RowsProcessed, src.RowsProcessed)
	atomic.AddUint64(&qs.RowsFound, src.RowsFound)
	atomic.AddUint64(&qs.ValuesRead, src.ValuesRead)
	atomic.AddUint64(&qs.TimestampsRead, src.TimestampsRead)
	atomic.AddUint64(&qs.BytesProcessedUncompressedValues, src.BytesProcessedUncompressedValues)

	if !qs.DetailedProfilingEnabled() {
		return
	}
	atomic.AddUint64(&qs.PartitionsTotal, src.PartitionsTotal)
	atomic.AddUint64(&qs.PartitionsSelected, src.PartitionsSelected)
	atomic.AddUint64(&qs.PartitionsTimeSkipped, src.PartitionsTimeSkipped)
	atomic.AddUint64(&qs.PartsTotal, src.PartsTotal)
	atomic.AddUint64(&qs.PartsSelected, src.PartsSelected)
	atomic.AddUint64(&qs.PartsTimeSkipped, src.PartsTimeSkipped)
	atomic.AddUint64(&qs.IndexBlockHeadersConsidered, src.IndexBlockHeadersConsidered)
	atomic.AddUint64(&qs.IndexBlockHeadersRead, src.IndexBlockHeadersRead)
	atomic.AddUint64(&qs.IndexBlockHeadersTimeOrKeySkipped, src.IndexBlockHeadersTimeOrKeySkipped)
	atomic.AddUint64(&qs.BlockHeadersDecoded, src.BlockHeadersDecoded)
	atomic.AddUint64(&qs.BlockHeadersTimeOrKeySkipped, src.BlockHeadersTimeOrKeySkipped)
	atomic.AddUint64(&qs.BlocksScheduled, src.BlocksScheduled)
	atomic.AddUint64(&qs.BlocksCancelledBeforeProcess, src.BlocksCancelledBeforeProcess)
	atomic.AddUint64(&qs.RowsSkippedBeforeScheduling, src.RowsSkippedBeforeScheduling)
}

// UpdateAtomicFromDataBlock adds query stats from db to qs.
func (qs *QueryStats) UpdateFromDataBlock(db *DataBlock) error {
	rowsCount := db.RowsCount()
	if rowsCount != 1 {
		return fmt.Errorf("unexpected number of rows in the query stats block; got %d; want 1", rowsCount)
	}

	var errGlobal error
	getUint64Entry := func(name string) uint64 {
		c := db.GetColumnByName(name)
		if c == nil {
			if errGlobal == nil {
				errGlobal = fmt.Errorf("cannot find field %q in query stats received from the remote storage", name)
			}
			return 0
		}
		v := c.Values[0]
		n, _ := tryParseUint64(v)
		return n
	}

	getOptionalUint64Entry := func(name string) uint64 {
		c := db.GetColumnByName(name)
		if c == nil {
			return 0
		}
		v := c.Values[0]
		n, _ := tryParseUint64(v)
		return n
	}

	qs.BytesReadColumnsHeaders += getUint64Entry("BytesReadColumnsHeaders")
	qs.BytesReadColumnsHeaderIndexes += getUint64Entry("BytesReadColumnsHeaderIndexes")
	qs.BytesReadBloomFilters += getUint64Entry("BytesReadBloomFilters")
	qs.BytesReadValues += getUint64Entry("BytesReadValues")
	qs.BytesReadTimestamps += getUint64Entry("BytesReadTimestamps")
	qs.BytesReadBlockHeaders += getUint64Entry("BytesReadBlockHeaders")

	qs.BlocksProcessed += getUint64Entry("BlocksProcessed")
	qs.RowsProcessed += getUint64Entry("RowsProcessed")
	qs.RowsFound += getUint64Entry("RowsFound")
	qs.ValuesRead += getUint64Entry("ValuesRead")
	qs.TimestampsRead += getUint64Entry("TimestampsRead")
	qs.BytesProcessedUncompressedValues += getUint64Entry("BytesProcessedUncompressedValues")

	if !qs.DetailedProfilingEnabled() {
		return errGlobal
	}
	qs.PartitionsTotal += getOptionalUint64Entry("PartitionsTotal")
	qs.PartitionsSelected += getOptionalUint64Entry("PartitionsSelected")
	qs.PartitionsTimeSkipped += getOptionalUint64Entry("PartitionsTimeSkipped")
	qs.PartsTotal += getOptionalUint64Entry("PartsTotal")
	qs.PartsSelected += getOptionalUint64Entry("PartsSelected")
	qs.PartsTimeSkipped += getOptionalUint64Entry("PartsTimeSkipped")
	qs.IndexBlockHeadersConsidered += getOptionalUint64Entry("IndexBlockHeadersConsidered")
	qs.IndexBlockHeadersRead += getOptionalUint64Entry("IndexBlockHeadersRead")
	qs.IndexBlockHeadersTimeOrKeySkipped += getOptionalUint64Entry("IndexBlockHeadersTimeOrKeySkipped")
	qs.BlockHeadersDecoded += getOptionalUint64Entry("BlockHeadersDecoded")
	qs.BlockHeadersTimeOrKeySkipped += getOptionalUint64Entry("BlockHeadersTimeOrKeySkipped")
	qs.BlocksScheduled += getOptionalUint64Entry("BlocksScheduled")
	qs.BlocksCancelledBeforeProcess += getOptionalUint64Entry("BlocksCancelledBeforeProcess")
	qs.RowsSkippedBeforeScheduling += getOptionalUint64Entry("RowsSkippedBeforeScheduling")

	return errGlobal
}

// CreateDataBlock creates a DataBlock from qs, including data-layout selection counters.
func (qs *QueryStats) CreateDataBlock(queryDurationNsecs int64) *DataBlock {
	return qs.createDataBlock(queryDurationNsecs, true)
}

// CreateLegacyDataBlock creates the legacy DataBlock used by v5 cluster protocols.
func (qs *QueryStats) CreateLegacyDataBlock(queryDurationNsecs int64) *DataBlock {
	return qs.createDataBlock(queryDurationNsecs, false)
}

func (qs *QueryStats) createDataBlock(queryDurationNsecs int64, includeSelectionCounters bool) *DataBlock {
	var cs []BlockColumn

	addUint64Entry := func(name string, value uint64) {
		cs = append(cs, BlockColumn{
			Name: name,
			Values: []string{
				string(marshalUint64String(nil, value)),
			},
		})
	}

	if includeSelectionCounters {
		qs.addEntries(addUint64Entry, queryDurationNsecs)
	} else {
		qs.addLegacyEntries(addUint64Entry, queryDurationNsecs)
	}

	return &DataBlock{
		columns: cs,
	}
}

func (qs *QueryStats) writeToPipeProcessor(pp pipeProcessor, queryDurationNsecs int64) {
	var rcs []resultColumn

	var buf []byte
	addUint64Entry := func(name string, value uint64) {
		rcs = append(rcs, resultColumn{})
		rc := &rcs[len(rcs)-1]
		rc.name = name
		bufLen := len(buf)
		buf = marshalUint64String(buf, value)
		v := bytesutil.ToUnsafeString(buf[bufLen:])
		rc.addValue(v)
	}

	if qs.DetailedProfilingEnabled() {
		qs.addEntries(addUint64Entry, queryDurationNsecs)
	} else {
		qs.addLegacyEntries(addUint64Entry, queryDurationNsecs)
	}

	var br blockResult
	br.setResultColumns(rcs, 1)
	pp.writeBlock(0, &br)
}

func (qs *QueryStats) addEntries(addUint64Entry func(name string, value uint64), queryDurationNsecs int64) {
	qs.addBaseEntries(addUint64Entry)

	addUint64Entry("PartitionsTotal", qs.PartitionsTotal)
	addUint64Entry("PartitionsSelected", qs.PartitionsSelected)
	addUint64Entry("PartitionsTimeSkipped", qs.PartitionsTimeSkipped)
	addUint64Entry("PartsTotal", qs.PartsTotal)
	addUint64Entry("PartsSelected", qs.PartsSelected)
	addUint64Entry("PartsTimeSkipped", qs.PartsTimeSkipped)
	addUint64Entry("IndexBlockHeadersConsidered", qs.IndexBlockHeadersConsidered)
	addUint64Entry("IndexBlockHeadersRead", qs.IndexBlockHeadersRead)
	addUint64Entry("IndexBlockHeadersTimeOrKeySkipped", qs.IndexBlockHeadersTimeOrKeySkipped)
	addUint64Entry("BlockHeadersDecoded", qs.BlockHeadersDecoded)
	addUint64Entry("BlockHeadersTimeOrKeySkipped", qs.BlockHeadersTimeOrKeySkipped)
	addUint64Entry("BlocksScheduled", qs.BlocksScheduled)
	addUint64Entry("BlocksCancelledBeforeProcess", qs.BlocksCancelledBeforeProcess)
	addUint64Entry("RowsSkippedBeforeScheduling", qs.RowsSkippedBeforeScheduling)

	addUint64Entry("QueryDurationNsecs", uint64(queryDurationNsecs))
}

func (qs *QueryStats) addLegacyEntries(addUint64Entry func(name string, value uint64), queryDurationNsecs int64) {
	qs.addBaseEntries(addUint64Entry)
	addUint64Entry("QueryDurationNsecs", uint64(queryDurationNsecs))
}

func (qs *QueryStats) addBaseEntries(addUint64Entry func(name string, value uint64)) {
	addUint64Entry("BytesReadColumnsHeaders", qs.BytesReadColumnsHeaders)
	addUint64Entry("BytesReadColumnsHeaderIndexes", qs.BytesReadColumnsHeaderIndexes)
	addUint64Entry("BytesReadBloomFilters", qs.BytesReadBloomFilters)
	addUint64Entry("BytesReadValues", qs.BytesReadValues)
	addUint64Entry("BytesReadTimestamps", qs.BytesReadTimestamps)
	addUint64Entry("BytesReadBlockHeaders", qs.BytesReadBlockHeaders)

	addUint64Entry("BytesReadTotal", qs.GetBytesReadTotal())

	addUint64Entry("BlocksProcessed", qs.BlocksProcessed)
	addUint64Entry("RowsProcessed", qs.RowsProcessed)
	addUint64Entry("RowsFound", qs.RowsFound)
	addUint64Entry("ValuesRead", qs.ValuesRead)
	addUint64Entry("TimestampsRead", qs.TimestampsRead)
	addUint64Entry("BytesProcessedUncompressedValues", qs.BytesProcessedUncompressedValues)
}

// Subtract returns the non-negative field-wise difference between qs and before.
// It is useful for attributing cumulative request stats to a single physical query.
func (qs QueryStatsSnapshot) Subtract(before QueryStatsSnapshot) QueryStatsSnapshot {
	sub := func(after, previous uint64) uint64 {
		if after < previous {
			return 0
		}
		return after - previous
	}
	return QueryStatsSnapshot{
		BytesReadTotal:                    sub(qs.BytesReadTotal, before.BytesReadTotal),
		BytesReadColumnsHeaders:           sub(qs.BytesReadColumnsHeaders, before.BytesReadColumnsHeaders),
		BytesReadColumnsHeaderIndexes:     sub(qs.BytesReadColumnsHeaderIndexes, before.BytesReadColumnsHeaderIndexes),
		BytesReadBloomFilters:             sub(qs.BytesReadBloomFilters, before.BytesReadBloomFilters),
		BytesReadValues:                   sub(qs.BytesReadValues, before.BytesReadValues),
		BytesReadTimestamps:               sub(qs.BytesReadTimestamps, before.BytesReadTimestamps),
		BytesReadBlockHeaders:             sub(qs.BytesReadBlockHeaders, before.BytesReadBlockHeaders),
		BlocksProcessed:                   sub(qs.BlocksProcessed, before.BlocksProcessed),
		RowsProcessed:                     sub(qs.RowsProcessed, before.RowsProcessed),
		RowsFound:                         sub(qs.RowsFound, before.RowsFound),
		ValuesRead:                        sub(qs.ValuesRead, before.ValuesRead),
		TimestampsRead:                    sub(qs.TimestampsRead, before.TimestampsRead),
		BytesProcessedUncompressedValues:  sub(qs.BytesProcessedUncompressedValues, before.BytesProcessedUncompressedValues),
		PartitionsTotal:                   sub(qs.PartitionsTotal, before.PartitionsTotal),
		PartitionsSelected:                sub(qs.PartitionsSelected, before.PartitionsSelected),
		PartitionsTimeSkipped:             sub(qs.PartitionsTimeSkipped, before.PartitionsTimeSkipped),
		PartsTotal:                        sub(qs.PartsTotal, before.PartsTotal),
		PartsSelected:                     sub(qs.PartsSelected, before.PartsSelected),
		PartsTimeSkipped:                  sub(qs.PartsTimeSkipped, before.PartsTimeSkipped),
		IndexBlockHeadersConsidered:       sub(qs.IndexBlockHeadersConsidered, before.IndexBlockHeadersConsidered),
		IndexBlockHeadersRead:             sub(qs.IndexBlockHeadersRead, before.IndexBlockHeadersRead),
		IndexBlockHeadersTimeOrKeySkipped: sub(qs.IndexBlockHeadersTimeOrKeySkipped, before.IndexBlockHeadersTimeOrKeySkipped),
		BlockHeadersDecoded:               sub(qs.BlockHeadersDecoded, before.BlockHeadersDecoded),
		BlockHeadersTimeOrKeySkipped:      sub(qs.BlockHeadersTimeOrKeySkipped, before.BlockHeadersTimeOrKeySkipped),
		BlocksScheduled:                   sub(qs.BlocksScheduled, before.BlocksScheduled),
		BlocksCancelledBeforeProcess:      sub(qs.BlocksCancelledBeforeProcess, before.BlocksCancelledBeforeProcess),
		RowsSkippedBeforeScheduling:       sub(qs.RowsSkippedBeforeScheduling, before.RowsSkippedBeforeScheduling),
		DetailedProfilingEnabled:          qs.DetailedProfilingEnabled,
	}
}
