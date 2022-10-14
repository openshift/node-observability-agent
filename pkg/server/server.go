package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const loopback = "127.0.0.1"

var slog = logrus.WithField("module", "server")

// Config holds the parameters necessary for the HTTP server, and agent in general need to run
type Config struct {
	Port                 int
	Token                string
	CACerts              *x509.CertPool
	NodeIP               string
	StorageFolder        string
	CrioUnixSocket       string
	CrioPreferUnixSocket bool
}

// Start starts HTTP server with parameters in cfg structure
func Start(cfg Config) {
	router := setupRoutes(cfg)

	// Clients must use TLS 1.2 or higher
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	httpServer := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf("%s:%d", loopback, cfg.Port),
		TLSConfig:    tlsConfig,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 40 * time.Second,
	}

	slog.Infof("listening on http://%s:%d", loopback, cfg.Port)
	slog.Infof("targeting node %s", cfg.NodeIP)

	panic(httpServer.ListenAndServe())
}
