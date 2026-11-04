package apptest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/httputil"

	otelpb "github.com/VictoriaMetrics/VictoriaTraces/lib/protoparser/opentelemetry/pb"
)

// Client is used for interacting with the apps over the network.
type Client struct {
	httpCli *http.Client
}

// NewClient creates a new client.
func NewClient() *Client {
	return &Client{
		httpCli: &http.Client{
			Transport: httputil.NewTransport(true, "apptest_client"),
		},
	}
}

// CloseConnections closes client connections.
func (c *Client) CloseConnections() {
	c.httpCli.CloseIdleConnections()
}

// Get sends a HTTP GET request, returns
// the response body and status code to the caller.
func (c *Client) Get(t *testing.T, url string) (string, int) {
	t.Helper()
	return c.Do(t, http.MethodGet, url, "", nil)
}

// Post sends a HTTP POST request, returns
// the response body and status code to the caller.
func (c *Client) Post(t *testing.T, url, contentType string, data []byte) (string, int) {
	t.Helper()
	return c.Do(t, http.MethodPost, url, contentType, data)
}

// PostForm sends a HTTP POST request containing the POST-form data, returns
// the response body and status code to the caller.
func (c *Client) PostForm(t *testing.T, url string, data url.Values) (string, int) {
	t.Helper()
	return c.Post(t, url, "application/x-www-form-urlencoded", []byte(data.Encode()))
}

// Delete sends a HTTP DELETE request and returns the response body and status code
// to the caller.
func (c *Client) Delete(t *testing.T, url string) (string, int) {
	t.Helper()
	return c.Do(t, http.MethodDelete, url, "", nil)
}

// Do prepares a HTTP request, sends it to the server, receives the response
// from the server, returns the response body and status code to the caller.
func (c *Client) Do(t *testing.T, method, url, contentType string, data []byte) (string, int) {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("could not create a HTTP request: %v", err)
	}

	if len(contentType) > 0 {
		req.Header.Add("Content-Type", contentType)
	}
	res, err := c.httpCli.Do(req)
	if err != nil {
		t.Fatalf("could not send HTTP request: %v", err)
	}

	body := readAllAndClose(t, res.Body)

	return body, res.StatusCode
}

func (c *Client) Write(t *testing.T, address string, data []string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("cannot dial %s: %s", address, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	d := []byte(strings.Join(data, "\n"))
	n, err := conn.Write(d)
	if err != nil {
		t.Fatalf("cannot write %d bytes to %s: %s", len(d), address, err)
	}
	if n != len(d) {
		t.Fatalf("BUG: conn.Write() returned unexpected number of written bytes to %s; got %d; want %d", address, n, len(d))
	}
}

// readAllAndClose reads everything from the response body and then closes it.
func readAllAndClose(t *testing.T, responseBody io.ReadCloser) string {
	t.Helper()

	defer responseBody.Close()
	b, err := io.ReadAll(responseBody)
	if err != nil {
		t.Fatalf("could not read response body: %d", err)
	}
	return string(b)
}

// ServesMetrics is used to retrieve the app's metrics.
//
// This type is expected to be embedded by the apps that serve metrics.
type ServesMetrics struct {
	metricsURL string
	cli        *Client
}

// GetIntMetric retrieves the value of a metric served by an app at /metrics URL.
// The value is then converted to int.
func (app *ServesMetrics) GetIntMetric(t *testing.T, metricName string) int {
	t.Helper()

	return int(app.GetMetric(t, metricName))
}

// GetMetric retrieves the value of a metric served by an app at /metrics URL.
func (app *ServesMetrics) GetMetric(t *testing.T, metricName string) float64 {
	t.Helper()

	value, err := app.TryGetMetric(t, metricName)
	if err != nil {
		t.Fatalf("get metric error: %s, %v", metricName, err)
	}
	return value
}

func (app *ServesMetrics) TryGetMetric(t *testing.T, metricName string) (float64, error) {
	t.Helper()

	metrics, statusCode := app.cli.Get(t, app.metricsURL)
	if statusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: got %d, want %d", statusCode, http.StatusOK)
	}
	for _, metric := range strings.Split(metrics, "\n") {
		value, found := strings.CutPrefix(metric, metricName)
		if found {
			value = strings.Trim(value, " ")
			res, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse metric value %s: %v", metric, err)
			}
			return res, nil
		}
	}
	return 0, fmt.Errorf("metric not found: %s", metricName)
}

// GetMetricsByPrefix retrieves the values of all metrics that start with given
// prefix.
func (app *ServesMetrics) GetMetricsByPrefix(t *testing.T, prefix string) []float64 {
	t.Helper()

	values := []float64{}

	metrics, statusCode := app.cli.Get(t, app.metricsURL)
	if statusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", statusCode, http.StatusOK)
	}
	for _, metric := range strings.Split(metrics, "\n") {
		if !strings.HasPrefix(metric, prefix) {
			continue
		}

		parts := strings.Split(metric, " ")
		if len(parts) < 2 {
			t.Fatalf("unexpected record format: got %q, want metric name and value separated by a space", metric)
		}

		value, err := strconv.ParseFloat(parts[len(parts)-1], 64)
		if err != nil {
			t.Fatalf("could not parse metric value %s: %v", metric, err)
		}

		values = append(values, value)
	}
	return values
}

var (
	http2Client *http.Client
	once        sync.Once
)

func GetHTTP2Client() *http.Client {
	once.Do(func() {
		var protocols http.Protocols
		protocols.SetUnencryptedHTTP2(true)
		http2Client = &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,
				Protocols:         &protocols,
			},
		}
	})
	return http2Client
}

// otlpGRPCExportTraces sends request as an `Export` gRPC call to url and returns the grpc-status of the response.
//
// gRPC errors are returned with HTTP status 200, so the caller must check grpc-status.
func otlpGRPCExportTraces(t *testing.T, url string, request *otelpb.ExportTraceServiceRequest) string {
	t.Helper()

	pbData := request.MarshalProtobuf(nil)

	// 5 bytes prefix: 1 byte compress flag + 4 bytes body length
	buf := make([]byte, 5)
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(pbData)))

	reqBody := append(buf, pbData...)

	// must use a http2 client
	client := GetHTTP2Client()

	resp, err := client.Post(url, "application/grpc", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("go error: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("got %d, expected 200", resp.StatusCode)
	}

	// trailers are available only after the body is read
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("cannot read response body: %s", err)
	}
	status := resp.Trailer.Get("grpc-status")
	if status == "" {
		status = resp.Header.Get("grpc-status")
	}
	if status == "" {
		t.Fatalf("missing grpc-status in the response")
	}
	return status
}
