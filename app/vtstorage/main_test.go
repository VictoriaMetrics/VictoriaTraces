package vtstorage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fs"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage/common"
)

func TestRunQueryOutOfRetention(t *testing.T) {
	// Create the test storage
	storagePath := t.Name()
	cfg := &logstorage.StorageConfig{
		Retention: 7 * 24 * time.Hour,
	}
	localStorage = logstorage.MustOpenStorage(storagePath, cfg)
	defer func() {
		// Close and delete the test storage
		localStorage.MustClose()
		fs.MustRemoveDir(storagePath)
	}()

	query, _ := logstorage.ParseQuery("*")
	// add a time filter which within the default retention period (7d).
	query.AddTimeFilter(0, time.Now().Add(-retentionPeriod.Duration()+5*time.Second).UnixNano())

	// the query should be executed with empty result.
	qctx := logstorage.NewQueryContext(context.TODO(), &logstorage.QueryStats{}, []logstorage.TenantID{}, query, false)
	if err := RunQuery(qctx, func(workerID uint, db *logstorage.DataBlock) {}); err != nil {
		t.Fatalf("RunQuery returns error for correct query")
	}

	// add a time filter which obviously out of the default retention period (7d).
	query.AddTimeFilter(0, time.Now().Add(-retentionPeriod.Duration()-10*time.Second).UnixNano())

	// the query should stop with ErrOutOfRetention error
	qctx = logstorage.NewQueryContext(context.TODO(), &logstorage.QueryStats{}, []logstorage.TenantID{}, query, false)
	if !errors.Is(RunQuery(qctx, nil), common.ErrOutOfRetention) {
		t.Fatalf("RunQuery fail to returns ErrOutOfRetention for query which with too small endTimestamp")
	}
}
