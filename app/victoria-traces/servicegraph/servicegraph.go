package servicegraph

import (
	"context"
	"flag"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	vtinsert "github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	vtselect "github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

var (
	enableServiceGraphTask     = flag.Bool("servicegraph.enableTask", false, "Whether to enable background task for generating service graph. It should only be enabled on VictoriaTraces single-node or vtstorage.")
	serviceGraphTaskInterval   = flag.Duration("servicegraph.taskInterval", time.Minute, "The background task interval for generating service graph data. It requires setting -servicegraph.enableTask=true.")
	serviceGraphTaskTimeout    = flag.Duration("servicegraph.taskTimeout", 30*time.Second, "The background task timeout duration for generating service graph data. It requires setting -servicegraph.enableTask=true.")
	serviceGraphTaskLookbehind = flag.Duration("servicegraph.taskLookbehind", time.Minute, "The lookbehind window for each time service graph background task run. It requires setting -servicegraph.enableTask=true.")
	serviceGraphTaskLimit      = flag.Uint64("servicegraph.taskLimit", 1000, "How many service graph relations each task could fetch for each tenant. It requires setting -servicegraph.enableTask=true.")
)

var (
	sgt *serviceGraphTask
)

func Init() {
	if *enableServiceGraphTask {
		sgt = newServiceGraphTask()
		sgt.Start()
	}
}

func Stop() {
	if *enableServiceGraphTask {
		sgt.Stop()
	}
}

type serviceGraphTask struct {
	stopCh chan struct{}
}

func newServiceGraphTask() *serviceGraphTask {
	return &serviceGraphTask{
		stopCh: make(chan struct{}),
	}
}

func (sgt *serviceGraphTask) Start() {
	logger.Infof("starting servicegraph background task, interval: %v, lookbehind: %v", *serviceGraphTaskInterval, *serviceGraphTaskLookbehind)
	go func() {
		ticker := time.NewTicker(*serviceGraphTaskInterval)
		defer ticker.Stop()

		for {
			select {
			case <-sgt.stopCh:
				return
			case <-ticker.C:
				ctx, cancelFunc := context.WithTimeout(context.Background(), *serviceGraphTaskTimeout)
				GenerateServiceGraphTimeRange(ctx)
				cancelFunc()
			}
		}
	}()
}

func (sgt *serviceGraphTask) Stop() {
	close(sgt.stopCh)
}

func GenerateServiceGraphTimeRange(ctx context.Context) {
	endTime := time.Now().Truncate(*serviceGraphTaskInterval)
	startTime := endTime.Add(-*serviceGraphTaskLookbehind)

	tenantIDs, err := vtstorage.GetTenantIDs(ctx, startTime.UnixNano(), endTime.UnixNano())
	if err != nil {
		logger.Errorf("cannot get tenant ids: %s", err)
		return
	}

	// query and persist operations are executed sequentially, which helps not to consume excessive resources.
	for _, tenantID := range tenantIDs {
		// query service graph relations
		rows, err := vtselect.GetServiceGraphTimeRange(ctx, tenantID, startTime, endTime, *serviceGraphTaskLimit)
		if err != nil {
			logger.Errorf("cannot get service graph for time range [%d, %d]: %s", startTime.Unix(), endTime.Unix(), err)
			return
		}
		if len(rows) == 0 {
			return
		}

		// persist service graph relations
		err = vtinsert.PersistServiceGraph(ctx, tenantID, rows, endTime)
		if err != nil {
			logger.Errorf("cannot persist service graph for time range [%d, %d]: %s", startTime.Unix(), endTime.Unix(), err)
		}
	}
}
