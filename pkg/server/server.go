package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

const loopback = "127.0.0.1"

var slog = logrus.WithField("module", "server")

// Config holds the parameters necessary for the HTTP server, and agent in general need to run
type Config struct {
	Port                 int
	UnixSocket           string
	PreferUnixSocket     bool
	Token                string
	CACerts              *x509.CertPool
	NodeIP               string
	StorageFolder        string
	CrioUnixSocket       string
	CrioPreferUnixSocket bool
	Mode                 string
}

// Start starts HTTP server with parameters in cfg structure
func Start(cfg Config) error {
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

	network := "tcp"
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(cfg.Port))
	if cfg.PreferUnixSocket {
		network = "unix"
		addr = cfg.UnixSocket
	}

	// gracefully shutdown HTTP server
	idleConnClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
		slog.Info("Received signal: ", <-sigCh)
		if err := httpServer.Shutdown(context.Background()); err != nil {
			slog.Errorf("Failed to gracefully shut down the server: %s", err.Error())
		}
		close(sigCh)
		close(idleConnClosed)
	}()

	slog.Infof("Start listening on %s://%s", network, addr)
	slog.Infof("Targeting node %s", cfg.NodeIP)

	ln, err := net.Listen(network, addr)
	if err != nil {
		return fmt.Errorf("failed on listen: %w", err)
	}
	if err := httpServer.Serve(ln); err != http.ErrServerClosed {
		return fmt.Errorf("failed on serve: %w", err)
	}

	<-idleConnClosed

	return nil
}
