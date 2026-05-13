package http2server

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/fs"
)

const (
	certFile = "test_http2_with_tls.crt"
	keyFile  = "test_http2_with_tls.key"

	testResponse = "testH2TLSResp"
)

func TestHTTP2WithTLS(t *testing.T) {
	addr := "127.0.0.1:10999"
	createHTTP2ServerWithTLS(t, addr)
	defer stopHTTP2ServerWithTLS(t, addr)

	// wait for the http server to start
	time.Sleep(200 * time.Millisecond)

	// setup HTTP/2 client
	client := http.DefaultClient

	tr := http.DefaultTransport.(*http.Transport).Clone()
	// we use self-gen cert so it's not secure and require skipping the check.
	tr.TLSClientConfig.InsecureSkipVerify = true
	tr.TLSClientConfig.NextProtos = []string{"h2"}

	client.Transport = tr

	// verify the test server with TLS
	resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
	if err != nil {
		t.Fatal("get HTTP/2 TLS server failed:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("incorrect HTTP/2 TLS server status code: got %d; want %d", resp.StatusCode, http.StatusOK)
	}

	if resp.Proto != "HTTP/2.0" {
		t.Fatalf("incorrect HTTP/2 TLS server proto: got %q; want %q", resp.Proto, "HTTP/2.0")
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != testResponse {
		t.Fatalf("incorrect HTTP/2 TLS server response body: got %q; want %q", body, testResponse)
	}
}

func createHTTP2ServerWithTLS(t *testing.T, addr string) {
	testHTTPHandler := func(w http.ResponseWriter, r *http.Request) bool {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testResponse))
		return true
	}

	createTLSFiles(certFile, keyFile)

	tlsCfg, err := NewTLSConfig(certFile, keyFile, "TLS13", nil)
	if err != nil {
		t.Fatalf("unexpected error when creating TLS config: %v", err)
		return
	}

	Serve(addr, testHTTPHandler, tlsCfg)
}

func stopHTTP2ServerWithTLS(t *testing.T, addr string) {
	if err := Stop([]string{addr}); err != nil {
		t.Fatalf("unexpected error when stopping HTTP2 server: %v", err)
	}
	removeTLSFiles(certFile, keyFile)
}

func createTLSFiles(certFile, keyFile string) {
	fs.MustWriteSync(certFile, []byte(testCRT))
	fs.MustWriteSync(keyFile, []byte(testPK))
}

func removeTLSFiles(certFile, keyFile string) {
	fs.MustRemovePath(certFile)
	fs.MustRemovePath(keyFile)
}

const (
	// all these keys are for testing and generated locally.
	testCRT = `-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIUNXxXTLjV8KQH73Rd/bP+1Qu7yxQwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQ04xEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNjA0MTQwNTQwMzBaFw0yNjA1
MTQwNTQwMzBaMEUxCzAJBgNVBAYTAkNOMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCcGlfxRuB2/1Me2+KIB7rHHJ5u2xA5uTZF8hzZ15SH
NO+XN9JkGKCHLMVseKkMbtZKsT05mKAyf4WPst/Z4vzJKK2v9qLp8x4/Bx3Uk3ms
ilZZ/8xHQjk6OsOjyVtrUBYi9lpl+yXCJZIxzQRxFrqv76+pxikuDCzDcPW81waI
VjnVm71HvXP8o52GspLwVqiSwn2fNvu6DqUOKh0zzwXrOLgBt/hunY9PWUpF01wf
660AZQydkqtE4h95wMoTi8EvRNVX0fttRAHSLuX2y/dp6/FONffCNmnRtPrcI4T3
n7zPaFtE9WWK/tc5d6FX0zLG2+RIOhVSZApGXHr56Uh2U/p+i25hQzCI6wDcoscu
PrMNUqef2Myyf0SlzwHlaV1l2FaZyKoE3dUn3vt5DOksHw+6nqp+yqw8a96PUfGE
hr3zcSGVglYzFCvPu9LwGvKANZ/W0d83jQ5I4ihT++vXHR5d3dbRfsXJoWlBo6TN
p8MfBiri2AOwDpzcXhs7uePEPES/yyiI68Hl/lN4iHCleYwpPQTpXBxSRCdB+d/s
tKDWx6H5LW0qDdVi8nJ+FRGYM48KnfDPQvJt7qFbldF/VdtBryzp2whovgPuzWTe
RewEHK6X3b3qVsy1L/IRIcOrZb+lg5RTHD2S8F0thqFmuCQSYiAB/VU1Zrc7dVRk
EQIDAQABo1MwUTAdBgNVHQ4EFgQURbB5zkDgLejbQQzwC0r8nef5YxUwHwYDVR0j
BBgwFoAURbB5zkDgLejbQQzwC0r8nef5YxUwDwYDVR0TAQH/BAUwAwEB/zANBgkq
hkiG9w0BAQsFAAOCAgEAe7vT8DIlyOPgtd5KDQn8adOdpuKPiGpvTZEQIzb2WD0l
Hpvw3nGstocViJa7OBmlGJ0pZvecuJISp/70BI1qL8NSnnx+vfpTRzBMrsZhW+ab
gZBaxbw35ihOqYO7vWcd7QxEK/l/VcPzKYwwrBWiwJJ0w7sQ8kDw1QOemswZAi3g
/3lk8xLlXOqYg6QoYuzUYsGpD/iyQ6vKsTQ/bfOf2rZ6lNq6V4SomYTYe/PayB5n
rMPSxPYJXkYCgWB5uZ5hmzubThKzER/TO170qEMRUjzErACELlFx17abfSbD/QuV
JYQAbJYEE6ZsJ6hEGLCc/qwcBp7apwzvBopWsmIMdAw1a8vqVvH044+HV9r++Ldj
sKWi76P3/LmiETCLP03GV0QLS93CjxVmWUUdYv1hOUiwldN/FREhuVzU3LWzDhAc
s6xHSoTygp2qF7YMFxZZ7TJEjR5GQHwOSFgN4Tvo6npSPl472WPXNoJGd9WIlNRi
w/tM1iyrdjPXEVeo47RjmwpSO4sSvYtct9Y/128Cs/8FT0rK0olXYnBahuw9uj9O
2GT7iWMX9DL8RtfJPzNTx6TOlUD7ocrI5mEMzoHj2JBkc6fqVOhYhCAenyhWRxfi
/0F2PFsp9E7bgZOF7k2LTSETGiKSKebtuphXYIhbK+pneZ9K4CkGp1Imd67QrtQ=
-----END CERTIFICATE-----`

	testPK = `-----BEGIN PRIVATE KEY-----
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
g8L4NFNh+Sz+4FX448t4PkhTulTSgw==
-----END PRIVATE KEY-----`
)
