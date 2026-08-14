package logstorage

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

// QueryProfileSnapshot is a point-in-time, JSON-friendly query profile.
type QueryProfileSnapshot struct {
	Queries       []QueryProfileQuery         `json:"queries"`
	QueryStats    QueryStatsSnapshot          `json:"query_stats"`
	NodeScanStats []QueryProfileNodeScanStats `json:"node_scan_stats"`
	Error         string                      `json:"error,omitempty"`
}

// QueryProfileNodeScanStats contains exact request-level scan stats returned by one storage node.
type QueryProfileNodeScanStats struct {
	Node       QueryProfileNode   `json:"node"`
	Stage      string             `json:"stage,omitempty"`
	Query      string             `json:"query,omitempty"`
	QueryStats QueryStatsSnapshot `json:"query_stats"`
	Error      string             `json:"error,omitempty"`
}

// QueryProfileQuery describes one physical query execution.
type QueryProfileQuery struct {
	ID        string            `json:"id"`
	Stage     string            `json:"stage,omitempty"`
	Scope     string            `json:"scope,omitempty"`
	Query     string            `json:"query"`
	Node      *QueryProfileNode `json:"node,omitempty"`
	StartedAt time.Time         `json:"started_at"`

	DurationNsecs int64  `json:"duration_nsecs"`
	Completed     bool   `json:"completed"`
	Error         string `json:"error,omitempty"`

	Operators  []QueryProfileOperator `json:"operators"`
	QueryStats QueryStatsSnapshot     `json:"query_stats"`
}

// QueryProfileNode identifies a storage node as assigned by the query coordinator.
type QueryProfileNode struct {
	Address string `json:"address"`
	Index   int    `json:"index"`
}

// QueryProfileOperator describes one operator in physical pipeline order.
type QueryProfileOperator struct {
	ID        string `json:"id"`
	Order     int    `json:"order"`
	Name      string `json:"name"`
	Synthetic bool   `json:"synthetic,omitempty"`

	InputBlocks  uint64 `json:"input_blocks"`
	InputRows    uint64 `json:"input_rows"`
	OutputBlocks uint64 `json:"output_blocks"`
	OutputRows   uint64 `json:"output_rows"`

	// InclusiveWriteDurationNsecs includes time spent in downstream operators called by writeBlock.
	// It is wall-clock duration, not CPU time or self time.
	InclusiveWriteDurationNsecs int64 `json:"inclusive_write_duration_nsecs"`

	// InclusiveFlushDurationNsecs includes synchronous downstream output generated while flushing.
	// It is wall-clock duration, not CPU time or self time.
	InclusiveFlushDurationNsecs int64 `json:"inclusive_flush_duration_nsecs"`

	// DownstreamDurationNsecs is synchronous time spent forwarding output to downstream operators.
	DownstreamDurationNsecs int64 `json:"downstream_duration_nsecs"`

	// ExclusiveActiveDurationNsecs is inclusive write+flush activity minus synchronous downstream forwarding.
	// It is summed wall-clock activity across workers, not CPU time.
	ExclusiveActiveDurationNsecs int64 `json:"exclusive_active_duration_nsecs"`

	Error string `json:"error,omitempty"`
}

// QueryProfileCollector concurrently collects physical query profiles.
// A nil collector disables profiling.
type QueryProfileCollector struct {
	mu            sync.Mutex
	nextID        uint64
	entries       []queryProfileEntry
	queryStats    *QueryStats
	nodeScanStats []QueryProfileNodeScanStats
}

type queryProfileEntry struct {
	live     *queryProfileQueryLive
	imported *QueryProfileQuery
}

// NewQueryProfileCollector returns an empty query profile collector.
func NewQueryProfileCollector() *QueryProfileCollector {
	return &QueryProfileCollector{}
}

func (qpc *QueryProfileCollector) attachQueryStats(qs *QueryStats) {
	if qpc == nil {
		return
	}
	qpc.mu.Lock()
	qpc.queryStats = qs
	qpc.mu.Unlock()
}

// Snapshot returns a concurrency-safe point-in-time copy of the collected profile,
// including aggregate QueryStats and exact per-node scan stats when available.
func (qpc *QueryProfileCollector) Snapshot() QueryProfileSnapshot {
	if qpc == nil {
		return QueryProfileSnapshot{
			Queries:       []QueryProfileQuery{},
			NodeScanStats: []QueryProfileNodeScanStats{},
		}
	}

	qpc.mu.Lock()
	entries := append([]queryProfileEntry(nil), qpc.entries...)
	queryStats := qpc.queryStats
	nodeScanStats := make([]QueryProfileNodeScanStats, len(qpc.nodeScanStats))
	copy(nodeScanStats, qpc.nodeScanStats)
	qpc.mu.Unlock()

	queries := make([]QueryProfileQuery, 0, len(entries))
	for _, entry := range entries {
		if entry.live != nil {
			queries = append(queries, entry.live.snapshot())
			continue
		}
		q := cloneQueryProfileQuery(*entry.imported)
		queries = append(queries, q)
	}
	return QueryProfileSnapshot{
		Queries:       queries,
		QueryStats:    queryStats.Snapshot(),
		NodeScanStats: nodeScanStats,
	}
}

// MergeRemote merges a remote snapshot and its marker-1 scan stats, then assigns
// the trusted coordinator-side node identity to both.
func (qpc *QueryProfileCollector) MergeRemote(snapshot QueryProfileSnapshot, queryStats QueryStatsSnapshot, nodeAddress string, nodeIndex int, requestStage, requestQuery string) {
	if qpc == nil {
		return
	}

	qpc.mu.Lock()
	defer qpc.mu.Unlock()
	node := QueryProfileNode{
		Address: nodeAddress,
		Index:   nodeIndex,
	}
	qpc.nodeScanStats = append(qpc.nodeScanStats, QueryProfileNodeScanStats{
		Node:       node,
		Stage:      requestStage,
		Query:      requestQuery,
		QueryStats: queryStats,
		Error:      snapshot.Error,
	})
	appendRemote := func(src QueryProfileQuery) {
		qpc.nextID++
		q := cloneQueryProfileQuery(src)
		q.ID = fmt.Sprintf("query-%d", qpc.nextID)
		q.Node = &QueryProfileNode{
			Address: node.Address,
			Index:   node.Index,
		}
		for i := range q.Operators {
			q.Operators[i].ID = fmt.Sprintf("%s-operator-%d", q.ID, i)
			q.Operators[i].Order = i
		}
		qpc.entries = append(qpc.entries, queryProfileEntry{imported: &q})
	}
	for _, src := range snapshot.Queries {
		appendRemote(src)
	}
	if snapshot.Error != "" && len(snapshot.Queries) == 0 {
		appendRemote(QueryProfileQuery{
			Stage:     requestStage,
			Scope:     "remote",
			Query:     requestQuery,
			StartedAt: time.Now(),
			Completed: true,
			Error:     snapshot.Error,
			Operators: []QueryProfileOperator{},
		})
	}
}

// CreateDataBlock serializes snapshot as JSON in a one-row DataBlock.
func (qps *QueryProfileSnapshot) CreateDataBlock() (*DataBlock, error) {
	b, err := json.Marshal(qps)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal query profile: %w", err)
	}
	db := &DataBlock{}
	db.SetColumns([]BlockColumn{{
		Name:   "QueryProfileJSON",
		Values: []string{string(b)},
	}})
	return db, nil
}

// QueryProfileSnapshotFromDataBlock decodes a snapshot from a one-row DataBlock.
func QueryProfileSnapshotFromDataBlock(db *DataBlock) (QueryProfileSnapshot, error) {
	if db.RowsCount() != 1 {
		return QueryProfileSnapshot{}, fmt.Errorf("unexpected number of rows in query profile block; got %d; want 1", db.RowsCount())
	}
	c := db.GetColumnByName("QueryProfileJSON")
	if c == nil {
		return QueryProfileSnapshot{}, fmt.Errorf("missing QueryProfileJSON field in query profile block")
	}
	var snapshot QueryProfileSnapshot
	if err := json.Unmarshal([]byte(c.Values[0]), &snapshot); err != nil {
		return QueryProfileSnapshot{}, fmt.Errorf("cannot unmarshal query profile JSON: %w", err)
	}
	if snapshot.Queries == nil {
		snapshot.Queries = []QueryProfileQuery{}
	}
	if snapshot.NodeScanStats == nil {
		snapshot.NodeScanStats = []QueryProfileNodeScanStats{}
	}
	return snapshot, nil
}

func cloneQueryProfileQuery(src QueryProfileQuery) QueryProfileQuery {
	dst := src
	if src.Node != nil {
		node := *src.Node
		dst.Node = &node
	}
	dst.Operators = append([]QueryProfileOperator(nil), src.Operators...)
	return dst
}

type queryProfileQueryLive struct {
	id         string
	stage      string
	scope      string
	query      string
	startedAt  time.Time
	operators  []*queryProfileOperatorLive
	queryStats *QueryStats
	statsStart QueryStatsSnapshot

	durationNsecs atomic.Int64
	completed     atomic.Bool
	errMu         sync.Mutex
	err           string
}

type queryProfileOperatorLive struct {
	id        string
	order     int
	name      string
	synthetic bool

	inputBlocks  atomic.Uint64
	inputRows    atomic.Uint64
	outputBlocks atomic.Uint64
	outputRows   atomic.Uint64

	inclusiveWriteDurationNsecs atomic.Int64
	inclusiveFlushDurationNsecs atomic.Int64
	downstreamDurationNsecs     atomic.Int64
	errMu                       sync.Mutex
	err                         string
}

func (qpc *QueryProfileCollector) beginQuery(stage, scope, query string, operatorNames []string, synthetic bool, queryStats *QueryStats) *queryProfileQueryLive {
	qpc.mu.Lock()
	qpc.nextID++
	id := fmt.Sprintf("query-%d", qpc.nextID)
	q := &queryProfileQueryLive{
		id:         id,
		stage:      stage,
		scope:      scope,
		query:      query,
		startedAt:  time.Now(),
		operators:  make([]*queryProfileOperatorLive, len(operatorNames)),
		queryStats: queryStats,
		statsStart: queryStats.Snapshot(),
	}
	for i, name := range operatorNames {
		q.operators[i] = &queryProfileOperatorLive{
			id:        fmt.Sprintf("%s-operator-%d", id, i),
			order:     i,
			name:      name,
			synthetic: synthetic,
		}
	}
	qpc.entries = append(qpc.entries, queryProfileEntry{live: q})
	qpc.mu.Unlock()
	return q
}

func (q *queryProfileQueryLive) finish(err error) {
	q.durationNsecs.Store(time.Since(q.startedAt).Nanoseconds())
	if err != nil {
		q.errMu.Lock()
		q.err = err.Error()
		q.errMu.Unlock()
	}
	q.completed.Store(true)
}

func (q *queryProfileQueryLive) snapshot() QueryProfileQuery {
	operators := make([]QueryProfileOperator, len(q.operators))
	for i, op := range q.operators {
		operators[i] = op.snapshot()
	}
	q.errMu.Lock()
	errText := q.err
	q.errMu.Unlock()
	durationNsecs := q.durationNsecs.Load()
	if !q.completed.Load() {
		durationNsecs = time.Since(q.startedAt).Nanoseconds()
	}
	return QueryProfileQuery{
		ID:            q.id,
		Stage:         q.stage,
		Scope:         q.scope,
		Query:         q.query,
		StartedAt:     q.startedAt,
		DurationNsecs: durationNsecs,
		Completed:     q.completed.Load(),
		Error:         errText,
		Operators:     operators,
		QueryStats:    q.queryStats.Snapshot().Subtract(q.statsStart),
	}
}

func (op *queryProfileOperatorLive) snapshot() QueryProfileOperator {
	op.errMu.Lock()
	errText := op.err
	op.errMu.Unlock()
	writeDuration := op.inclusiveWriteDurationNsecs.Load()
	flushDuration := op.inclusiveFlushDurationNsecs.Load()
	downstreamDuration := op.downstreamDurationNsecs.Load()
	exclusiveDuration := writeDuration + flushDuration - downstreamDuration
	if exclusiveDuration < 0 {
		exclusiveDuration = 0
	}
	return QueryProfileOperator{
		ID:                           op.id,
		Order:                        op.order,
		Name:                         op.name,
		Synthetic:                    op.synthetic,
		InputBlocks:                  op.inputBlocks.Load(),
		InputRows:                    op.inputRows.Load(),
		OutputBlocks:                 op.outputBlocks.Load(),
		OutputRows:                   op.outputRows.Load(),
		InclusiveWriteDurationNsecs:  writeDuration,
		InclusiveFlushDurationNsecs:  flushDuration,
		DownstreamDurationNsecs:      downstreamDuration,
		ExclusiveActiveDurationNsecs: exclusiveDuration,
		Error:                        errText,
	}
}

func (op *queryProfileOperatorLive) setError(err error) {
	if err == nil {
		return
	}
	op.errMu.Lock()
	if op.err == "" {
		op.err = err.Error()
	}
	op.errMu.Unlock()
}

type queryProfilePipeProcessor struct {
	op *queryProfileOperatorLive
	pp pipeProcessor
}

func (pp *queryProfilePipeProcessor) writeBlock(workerID uint, br *blockResult) {
	pp.op.inputBlocks.Add(1)
	pp.op.inputRows.Add(uint64(br.rowsLen))
	startTime := time.Now()
	pp.pp.writeBlock(workerID, br)
	pp.op.inclusiveWriteDurationNsecs.Add(time.Since(startTime).Nanoseconds())
}

func (pp *queryProfilePipeProcessor) flush() error {
	startTime := time.Now()
	err := pp.pp.flush()
	pp.op.inclusiveFlushDurationNsecs.Add(time.Since(startTime).Nanoseconds())
	pp.op.setError(err)
	return err
}

type queryProfileOutputPipeProcessor struct {
	op     *queryProfileOperatorLive
	ppNext pipeProcessor
}

func (pp *queryProfileOutputPipeProcessor) writeBlock(workerID uint, br *blockResult) {
	pp.op.outputBlocks.Add(1)
	pp.op.outputRows.Add(uint64(br.rowsLen))
	startTime := time.Now()
	pp.ppNext.writeBlock(workerID, br)
	pp.op.downstreamDurationNsecs.Add(time.Since(startTime).Nanoseconds())
}

func (pp *queryProfileOutputPipeProcessor) flush() error {
	panic("BUG: queryProfileOutputPipeProcessor.flush must not be called")
}

func getPipeName(p pipe) string {
	t := reflect.TypeOf(p)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	runes := []rune(strings.TrimPrefix(t.Name(), "pipe"))
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prevIsLower := unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevIsLower || nextIsLower {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
