package apptest

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// Vtcluster holds the state of a VictoriaTraces cluster.
type Vtcluster struct {
	storageNodes []*Vtsingle
	insertNode   *Vtsingle
	selectNode   *Vtsingle
}

// StartVtcluster starts VictoriaTraces cluster with the given storageFlags for storage nodes.
// Each storage node keeps its data in its own directory under dataDir.
//
// Stop must be called on the returned Vtcluster when it is no longer needed.
func StartVtcluster(instance, dataDir string, storageFlags []string, cli *Client) (*Vtcluster, error) {
	app := &Vtcluster{}

	storageNodeAddrs := make([]string, 3)
	for i := range 3 {
		name := fmt.Sprintf("%s-storage-%d", instance, i)
		flags := append(slices.Clone(storageFlags), fmt.Sprintf("-storageDataPath=%s/%s", dataDir, name))
		node, err := StartVtsingle(name, flags, nil, cli)
		if err != nil {
			app.Stop()
			return nil, err
		}
		app.storageNodes = append(app.storageNodes, node)
		storageNodeAddrs[i] = node.httpListenAddr
	}
	storageNodeFlag := "-storageNode=" + strings.Join(storageNodeAddrs, ",")

	var err error
	app.insertNode, err = startVtnode(instance+"-insert", []string{storageNodeFlag, "-select.disable=true"}, cli, true)
	if err != nil {
		app.Stop()
		return nil, err
	}
	app.selectNode, err = startVtnode(instance+"-select", []string{storageNodeFlag, "-insert.disable=true"}, cli, false)
	if err != nil {
		app.Stop()
		return nil, err
	}
	return app, nil
}

// startVtnode starts a vtinsert or vtselect node, which has no local storage.
func startVtnode(instance string, flags []string, cli *Client, withGRPC bool) (*Vtsingle, error) {
	extractREs := []*regexp.Regexp{httpListenAddrRE}
	if withGRPC {
		extractREs = append(extractREs, gRPCListenAddrRE)
	}
	app, stderrExtracts, err := startApp(instance, "../../bin/victoria-traces-race", flags, &appOptions{
		defaultFlags: map[string]string{
			"-httpListenAddr":     "127.0.0.1:0",
			"-otlpGRPCListenAddr": "127.0.0.1:0",
			"-otlpGRPC.tls":       "false",
			// the same as for vtsingle, so the tests don't wait for the production defaults
			"-insert.indexFlushInterval": "2s",
			"-search.latencyOffset":      "2s",
		},
		extractREs: extractREs,
	})
	if err != nil {
		return nil, err
	}
	gRPCListenAddr := ""
	if withGRPC {
		gRPCListenAddr = stderrExtracts[1]
	}
	return newVtsingle(app, cli, "", stderrExtracts[0], gRPCListenAddr), nil
}

// Stop stops all the cluster nodes.
func (app *Vtcluster) Stop() {
	for _, node := range []*Vtsingle{app.insertNode, app.selectNode} {
		if node != nil {
			node.Stop()
		}
	}
	for _, node := range app.storageNodes {
		node.Stop()
	}
}

// ForceFlush flushes the inserted data on all the storage nodes via the insert node.
func (app *Vtcluster) ForceFlush(t *testing.T) {
	t.Helper()
	app.insertNode.ForceFlush(t)
}

// ForceMerge forces the merging of parts on all the storage nodes.
func (app *Vtcluster) ForceMerge(t *testing.T) {
	t.Helper()
	for _, node := range app.storageNodes {
		node.ForceMerge(t)
	}
}

// OTLPHTTPExportTraces sends the request to the insert node.
func (app *Vtcluster) OTLPHTTPExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts) {
	t.Helper()
	app.insertNode.OTLPHTTPExportTraces(t, request, opts)
}

// OTLPgRPCExportTraces sends the request to the insert node.
func (app *Vtcluster) OTLPgRPCExportTraces(t *testing.T, request *otelpb.ExportTraceServiceRequest, opts QueryOpts) {
	t.Helper()
	app.insertNode.OTLPgRPCExportTraces(t, request, opts)
}

// OTLPHTTPExportRawTraces sends the data to the insert node.
func (app *Vtcluster) OTLPHTTPExportRawTraces(t *testing.T, data []byte, opts QueryOpts) {
	t.Helper()
	app.insertNode.OTLPHTTPExportRawTraces(t, data, opts)
}

// JaegerAPIServices queries the select node.
func (app *Vtcluster) JaegerAPIServices(t *testing.T, opts QueryOpts) *JaegerAPIServicesResponse {
	t.Helper()
	return app.selectNode.JaegerAPIServices(t, opts)
}

// JaegerAPIOperations queries the select node.
func (app *Vtcluster) JaegerAPIOperations(t *testing.T, serviceName string, opts QueryOpts) *JaegerAPIOperationsResponse {
	t.Helper()
	return app.selectNode.JaegerAPIOperations(t, serviceName, opts)
}

// JaegerAPITraces queries the select node.
func (app *Vtcluster) JaegerAPITraces(t *testing.T, param JaegerQueryParam, opts QueryOpts) *JaegerAPITracesResponse {
	t.Helper()
	return app.selectNode.JaegerAPITraces(t, param, opts)
}

// JaegerAPITrace queries the select node.
func (app *Vtcluster) JaegerAPITrace(t *testing.T, traceID string, opts QueryOpts) *JaegerAPITraceResponse {
	t.Helper()
	return app.selectNode.JaegerAPITrace(t, traceID, opts)
}

// JaegerAPIDependencies queries the select node.
func (app *Vtcluster) JaegerAPIDependencies(t *testing.T, param JaegerDependenciesParam, opts QueryOpts) *JaegerAPIDependenciesResponse {
	t.Helper()
	return app.selectNode.JaegerAPIDependencies(t, param, opts)
}

// TempoAPISearch queries the select node.
func (app *Vtcluster) TempoAPISearch(t *testing.T, q string, start, end time.Time) *TempoAPISearchResponse {
	t.Helper()
	return app.selectNode.TempoAPISearch(t, q, start, end)
}

// LogsQLQuery queries the select node.
func (app *Vtcluster) LogsQLQuery(t *testing.T, LogsQL string, opts QueryOpts) *LogsQLQueryResponse {
	t.Helper()
	return app.selectNode.LogsQLQuery(t, LogsQL, opts)
}

// String returns the string representation of the app state.
func (app *Vtcluster) String() string {
	return "Vtcluster"
}
