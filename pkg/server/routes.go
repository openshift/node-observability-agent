package server

import (
	"github.com/gorilla/mux"

	"github.com/openshift/node-observability-agent/pkg/handlers"
)

func setupRoutes(cfg Config) *mux.Router {
	r := mux.NewRouter()
	if cfg.Mode == "profiling" {
		h := handlers.NewHandlers(cfg.Token, cfg.CACerts, cfg.StorageFolder, cfg.CrioUnixSocket, cfg.NodeIP, cfg.CrioPreferUnixSocket)
		r.HandleFunc("/node-observability-pprof", h.HandleProfiling)
		r.HandleFunc("/node-observability-status", h.Status)
	} else if cfg.Mode == "scripting" {
		h := handlers.NewScriptingHandlers(cfg.StorageFolder, cfg.NodeIP)
		r.HandleFunc("/node-observability-scripting", h.HandleScripting)
		r.HandleFunc("/node-observability-status", h.Status)
	}
	return r
}
