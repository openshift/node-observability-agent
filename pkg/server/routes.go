package server

import (
	"io/ioutil"

	"github.com/gorilla/mux"
	"github.com/sherine-k/node-observability-agent/pkg/handlers"
)

func setupRoutes(tokenFile string, storageFolder string, crioUnixSocket string) *mux.Router {
	token, err := readTokenFile(tokenFile)
	if err != nil {
		panic("Unable to read token file")
	}
	h := handlers.NewHandlers(token, storageFolder, crioUnixSocket)
	r := mux.NewRouter()
	r.HandleFunc("/crio/profiling", h.ProfileCrio)
	r.HandleFunc("/kubelet/profiling", h.ProfileKubelet)
	return r
}

func readTokenFile(tokenFile string) (string, error) {
	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
