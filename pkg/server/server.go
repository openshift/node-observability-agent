package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

var slog = logrus.WithField("module", "server")

type Config struct {
	Port          int
	TokenFile     string
	StorageFolder string
}

func Start(cfg Config) {
	router := setupRoutes(cfg.TokenFile, cfg.StorageFolder)

	// Clients must use TLS 1.2 or higher
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	httpServer := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		TLSConfig:    tlsConfig,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 40 * time.Second,
	}

	slog.Infof("listening on http://:%d", cfg.Port)
	panic(httpServer.ListenAndServe())

}
