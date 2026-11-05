package apptest

import (
	"fmt"
	"regexp"
	"strings"
)

// Vtselect holds the state of a victoria-traces app, which sends select queries
// to the -storageNode nodes instead of a local storage.
type Vtselect struct {
	*app
	*ServesMetrics

	httpListenAddr string
}

// StartVtselect starts an instance of victoria-traces, which sends select queries
// to the given storageNodes. It also sets the default flags and populates the app
// instance state with runtime values extracted from the application log (such
// as httpListenAddr).
func StartVtselect(instance string, storageNodes, flags, env []string, cli *Client) (*Vtselect, error) {
	app, stderrExtracts, err := startApp(instance, "../../bin/victoria-traces-race", flags, &appOptions{
		env: env,
		defaultFlags: map[string]string{
			"-httpListenAddr":     "127.0.0.1:0",
			"-otlpGRPCListenAddr": "127.0.0.1:0",
			"-otlpGRPC.tls":       "false",
			"-storageNode":        strings.Join(storageNodes, ","),
		},
		extractREs: []*regexp.Regexp{
			httpListenAddrRE,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Vtselect{
		app: app,
		ServesMetrics: &ServesMetrics{
			metricsURL: fmt.Sprintf("http://%s/metrics", stderrExtracts[0]),
			cli:        cli,
		},
		httpListenAddr: stderrExtracts[0],
	}, nil
}

// HTTPAddr returns the address at which the vtselect process is listening
// for http connections.
func (app *Vtselect) HTTPAddr() string {
	return app.httpListenAddr
}
