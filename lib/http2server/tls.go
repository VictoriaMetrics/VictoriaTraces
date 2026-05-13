package http2server

import (
	"crypto/tls"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/netutil"
)

// NewTLSConfig builds TLS config for HTTP/2 Server.
func NewTLSConfig(certFile, keyFile, minVersion string, cipherSuites []string) (*tls.Config, error) {
	cfg, err := netutil.GetServerTLSConfig(certFile, keyFile, minVersion, cipherSuites)
	if err != nil {
		return nil, err
	}

	// Allow using TLS with HTTP/2 by specifying NextProtos.
	// See https://datatracker.ietf.org/doc/html/rfc7540#section-3.3
	// See https://github.com/VictoriaMetrics/VictoriaTraces/issues/108
	cfg.NextProtos = []string{"h2"}
	return cfg, nil
}
