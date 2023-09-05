package server

import (
	"github.com/gorilla/mux"

	"github.com/openshift/node-observability-agent/pkg/handlers"
)

func setupRoutes(cfg Config) *mux.Router {

	r := mux.NewRouter()
	if cfg.Mode == "profile" {
		h := handlers.NewHandlers(cfg.Token, cfg.StorageFolder, cfg.CrioUnixSocket, cfg.NodeIP)
		r.HandleFunc("/node-observability-pprof", h.HandleProfiling)
		r.HandleFunc("/node-observability-status", h.Status)
	} else if cfg.Mode == "metrics" {
		h := handlers.NewMetricsHandlers(cfg.StorageFolder, cfg.NodeIP)
		r.HandleFunc("/node-observability-metrics", h.HandleMetrics)
		r.HandleFunc("/node-observability-status", h.Status)
	}
	return r
}
