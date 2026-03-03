package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/encoding/zstd"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
)

func TestServeSamplingDecision(t *testing.T) {
	rh := func(w http.ResponseWriter, r *http.Request) bool {
		switch r.URL.Path {
		case "/api/v1/remotesampling_decision":
			b, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Errorf("cannot read body: %s", err)
				return false
			}
			bb := make([]byte, 0, len(b))
			bb, err = zstd.Decompress(bb, b)
			if err != nil {
				logger.Errorf("cannot decompress body: %s", err)
				return false
			}

			if len(bb)%16 != 0 {
				logger.Errorf("unexpected length of bb: %d", len(bb))
				return false
			}

			for i := 0; i < len(bb); i += 16 {
				logger.Infof("sampled traceID: %v", bb[i:i+16])
			}

			return true
		}
		return false
	}
	go httpserver.Serve([]string{"0.0.0.0:10499"}, rh, httpserver.ServeOptions{})
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
}
