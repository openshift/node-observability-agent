package server

import (
	"github.com/gorilla/mux"

	"github.com/openshift/node-observability-agent/pkg/handlers"
)

func setupRoutes(cfg Config) *mux.Router {
	h := handlers.NewHandlers(cfg.Token, cfg.CACerts, cfg.StorageFolder, cfg.CrioUnixSocket, cfg.NodeIP, cfg.CrioPreferUnixSocket)
	r := mux.NewRouter()
	r.HandleFunc("/node-observability-pprof", h.HandleProfiling)
	r.HandleFunc("/node-observability-status", h.Status)
	return r
}
