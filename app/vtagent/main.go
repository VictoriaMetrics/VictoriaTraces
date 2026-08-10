package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/buildinfo"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/envflag"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/flagutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httpserver"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/logger"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/procutil"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/pushmetrics"

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtagent/remotewrite"
	vtinsert "github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
)

var (
	httpListenAddrs = flagutil.NewArrayString("httpListenAddr", "TCP address to listen for incoming http requests. "+
		"Set this flag to empty value in order to disable listening on any port. This mode may be useful for running multiple vtagent instances on the same server. "+
		"Note that /targets and /metrics pages aren't available if -httpListenAddr=''. See also -tls and -httpListenAddr.useProxyProtocol")
	useProxyProtocol = flagutil.NewArrayBool("httpListenAddr.useProxyProtocol", "Whether to use proxy protocol for connections accepted at the corresponding -httpListenAddr . "+
		"See https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt . "+
		"With enabled proxy protocol http server cannot serve regular /metrics endpoint. Use -pushmetrics.url for metrics pushing")
	tmpDataPath = flag.String("tmpDataPath", "", "Base directory for storing vtagent data. "+
		"Used as default for -remoteWrite.tmpDataPath, -kubernetesCollector.checkpointsPath, "+
		"and -fileCollector.checkpointsPath unless those flags are set explicitly")
)

func main() {
	// Write flags and help message to stdout, since it is easier to grep or pipe.
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = usage
	envflag.Parse()
	buildinfo.Init()
	initSecretFlags()
	logger.Init()

	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":10429"}
	}
	logger.Infof("starting vtagent at %q...", listenAddrs)
	startTime := time.Now()

	insertutil.SetLogRowsStorage(&remotewrite.Storage{})
	remotewrite.Init(*tmpDataPath)

	vtinsert.Init()

	go httpserver.Serve(listenAddrs, requestHandler, httpserver.ServeOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("started vtagent in %.3f seconds", time.Since(startTime).Seconds())

	pushmetrics.Init()
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
	pushmetrics.Stop()

	startTime = time.Now()
	logger.Infof("gracefully shutting down webservice at %q", listenAddrs)
	if err := httpserver.Stop(listenAddrs); err != nil {
		logger.Fatalf("cannot stop the webservice: %s", err)
	}
	vtinsert.Stop()
	remotewrite.Stop()
	logger.Infof("successfully shut down the webservice in %.3f seconds", time.Since(startTime).Seconds())
	logger.Infof("successfully stopped vtagent in %.3f seconds", time.Since(startTime).Seconds())
}

// RequestHandler handles insert requests for VictoriaTraces
func requestHandler(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			return false
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h2>vtagent</h2>")
		fmt.Fprintf(w, "See docs at <a href='https://docs.victoriametrics.com/victoriatraces/vtagent/'>https://docs.victoriametrics.com/victoriatraces/vtagent/</a></br>")
		fmt.Fprintf(w, "Useful endpoints:</br>")
		httpserver.WriteAPIHelp(w, [][2]string{
			{"metrics", "available service metrics"},
			{"flags", "command-line flags"},
		})
		return true
	}
	return vtinsert.RequestHandler(w, r)
}

func usage() {
	const s = `
vtagent collects trace spans via popular data ingestion protocols and routes it to VictoriaTraces.

See the docs at https://docs.victoriametrics.com/victoriatraces/vtagent/ .
`
	flagutil.Usage(s)
}

// initSecretFlags manage the default secret flags for vtagent application.
func initSecretFlags() {
	remotewrite.InitSecretFlags()
}
