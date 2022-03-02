package server

import (
	"github.com/gorilla/mux"
	"github.com/sherine-k/node-observability-agent/pkg/handlers"
)

func setupRoutes(cfg Config) *mux.Router {

	h := handlers.NewHandlers(cfg.Token, cfg.StorageFolder, cfg.CrioUnixSocket, cfg.NodeIP)
	r := mux.NewRouter()
	r.HandleFunc("/pprof", h.HandleProfiling)
	r.HandleFunc("/status", h.Status)
	return r
}
