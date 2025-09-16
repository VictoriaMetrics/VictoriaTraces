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

	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtinsert/insertutil"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtselect"
	"github.com/VictoriaMetrics/VictoriaTraces/app/vtstorage"
)

var (
	httpListenAddrs  = flagutil.NewArrayString("httpListenAddr", "TCP address to listen for incoming http requests. See also -httpListenAddr.useProxyProtocol")
	useProxyProtocol = flagutil.NewArrayBool("httpListenAddr.useProxyProtocol", "Whether to use proxy protocol for connections accepted at the given -httpListenAddr . "+
		"See https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt . "+
		"With enabled proxy protocol http server cannot serve regular /metrics endpoint. Use -pushmetrics.url for metrics pushing")
)

func main() {
	// Write flags and help message to stdout, since it is easier to grep or pipe.
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = usage
	envflag.Parse()
	buildinfo.Init()
	logger.Init()

	listenAddrs := *httpListenAddrs
	if len(listenAddrs) == 0 {
		listenAddrs = []string{":10428"}
	}
	logger.Infof("starting VictoriaTraces at %q...", listenAddrs)
	startTime := time.Now()

	vtstorage.Init()
	vtselect.Init()

	insertutil.SetLogRowsStorage(&vtstorage.Storage{})
	vtinsert.Init()

	go httpserver.Serve(listenAddrs, requestHandler, httpserver.ServeOptions{
		UseProxyProtocol: useProxyProtocol,
	})
	logger.Infof("started VictoriaTraces in %.3f seconds; see https://docs.victoriametrics.com/victoriatraces/", time.Since(startTime).Seconds())

	pushmetrics.Init()
	sig := procutil.WaitForSigterm()
	logger.Infof("received signal %s", sig)
	pushmetrics.Stop()

	logger.Infof("gracefully shutting down webservice at %q", listenAddrs)
	startTime = time.Now()
	if err := httpserver.Stop(listenAddrs); err != nil {
		logger.Fatalf("cannot stop the webservice: %s", err)
	}
	logger.Infof("successfully shut down the webservice in %.3f seconds", time.Since(startTime).Seconds())

	vtinsert.Stop()
	vtselect.Stop()
	vtstorage.Stop()

	logger.Infof("the VictoriaTraces has been stopped in %.3f seconds", time.Since(startTime).Seconds())
}

func requestHandler(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" {
		if r.Method != http.MethodGet {
			return false
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h2>Single-node VictoriaTraces</h2></br>")
		fmt.Fprintf(w, "Version %s<br>", buildinfo.Version)
		fmt.Fprintf(w, "See docs at <a href='https://docs.victoriametrics.com/victoriatraces/'>https://docs.victoriametrics.com/victoriatraces/</a></br>")
		fmt.Fprintf(w, "Useful endpoints:</br>")
		httpserver.WriteAPIHelp(w, [][2]string{
			{"select/vmui", "Web UI for VictoriaTraces"},
			{"metrics", "available service metrics"},
			{"flags", "command-line flags"},
		})
		return true
	}

	if vtinsert.RequestHandler(w, r) {
		return true
	}
	if vtselect.RequestHandler(w, r) {
		return true
	}
	if vtstorage.RequestHandler(w, r) {
		return true
	}
	return false
}

func usage() {
	const s = `
victoria-traces is a traces storage and analytics service.

See the docs at https://docs.victoriametrics.com/victoriatraces/
`
	flagutil.Usage(s)
}
