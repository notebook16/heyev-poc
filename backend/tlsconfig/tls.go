package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"heyev-backend-poc/config"
)

func Load(endpoint string) (*tls.Config, error) {
	caCert, err := os.ReadFile(config.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	cert, err := tls.LoadX509KeyPair(config.ClientCertFile, config.ClientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}

	return &tls.Config{
		RootCAs:            caPool,
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS12,
		ServerName:         endpoint,
		InsecureSkipVerify: false,
	}, nil
}
