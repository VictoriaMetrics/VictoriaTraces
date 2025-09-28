package backgroundtask

import (
	"context"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"

	vtinsert "github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/opentelemetry"
	vtselect "github.com/VictoriaMetrics/VictoriaTraces/app/vtselect/traces/query"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

var (
	enableServiceGraph      = flag.Bool("servicegraph.enable", false, "Whether to enable background task for generating service graph. It should only be enabled on VictoriaTraces single-node or vtstorage.")
	serviceGraphInterval    = flag.Duration("servicegraph.taskInterval", time.Minute, "The background task interval for generating service graph data. It requires setting `-servicegraph.enable=true`.")
	serviceGraphTaskTimeout = flag.Duration("servicegraph.taskTimeout", 30*time.Second, "The background task timeout duration for generating service graph data. It requires setting `-servicegraph.enable=true`.")
	serviceGraphLookbehind  = flag.Duration("servicegraph.lookbehind", time.Minute, "The lookbehind window for each time service graph background task run. It requires setting `-servicegraph.enable=true`.")
)

var (
	sgt *serviceGraphTask
)

func Init() {
	if *enableServiceGraph {
		sgt = newServiceGraphTask()
		sgt.Start()
	}
	return
}

func Stop() {
	if *enableServiceGraph {
		sgt.Stop()
	}
	return
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
	logger.Infof("starting background task for service graph, interval: %v, lookbehind: %v", *serviceGraphInterval, *serviceGraphLookbehind)
	ticker := time.NewTicker(*serviceGraphInterval)
	go func() {
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
	return
}

func (sgt *serviceGraphTask) Stop() {
	close(sgt.stopCh)
	return
}

func GenerateServiceGraphTimeRange(ctx context.Context) {
	endTime := time.Now().Truncate(*serviceGraphInterval)
	startTime := endTime.Add(-*serviceGraphLookbehind)

	tenantIDs, err := vtstorage.GetTenantIDsByTimeRange(ctx, startTime.UnixNano(), endTime.UnixNano())
	if err != nil {
		logger.Errorf("cannot get tenant ids: %s", err)
		return
	}

	// query and persist operations are executed sequentially, which helps not to consume excessive resources.
	for _, tenantID := range tenantIDs {
		r, _ := http.NewRequestWithContext(ctx, "", "", nil)
		r.Header.Set("AccountID", strconv.FormatUint(uint64(tenantID.AccountID), 10))
		r.Header.Set("ProjectID", strconv.FormatUint(uint64(tenantID.ProjectID), 10))
		rows, err := vtselect.GetServiceGraphTimeRange(ctx, r, startTime, endTime)
		if err != nil {
			logger.Errorf("cannot get service graph for time range [%d, %d]: %s", startTime.Unix(), endTime.Unix(), err)
			return
		}
		if len(rows) == 0 {
			return
		}

		err = vtinsert.PersistServiceGraph(ctx, r, rows, endTime)
		if err != nil {
			logger.Errorf("cannot presist service graph for time %d: %s", endTime.Unix(), err)
		}
	}

	return
}
