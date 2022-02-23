package handlers

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

var hlog = logrus.WithField("module", "handler")

type Handlers struct {
	Token         string
	NodeIP        string
	StorageFolder string
}

func NewHandlers(token string, storageFolder string) *Handlers {
	//Get env var NODE_IP
	node := os.Getenv("NODE_IP")
	if node == "" {
		panic("Did not find environment variable $NODE_IP")
	}
	return &Handlers{
		Token:         token,
		NodeIP:        node,
		StorageFolder: storageFolder,
	}
}

func (h *Handlers) ProfileCrio(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("OK"))
	if err != nil {
		hlog.Errorf("could not write response: %v", err)
	}
}

func (h *Handlers) ProfileKubelet(w http.ResponseWriter, r *http.Request) {
	//TODO Go back to securely making this request
	//Prepare http client that ignores tls check
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transCfg}

	//Construct HTTP Req
	req, err := http.NewRequest("GET", "https://"+h.NodeIP+":10250/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "Bearer "+h.Token)
	if err != nil {
		hlog.Errorf("Error preparing http request https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error preparing http request  https://" + h.NodeIP + ":10250/debug/pprof/profile"))
		hlog.Errorf("Error preparing request %v", err)
		return
	}
	//Handle HTTP Req
	res, err := client.Do(req)
	if err != nil {
		hlog.Errorf("Error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error with HTTP request for kubelet profiling https://" + h.NodeIP + ":10250/debug/pprof/profile"))
		hlog.Errorf("Error writing response to profiling request %v", err)
		return
	}

	defer res.Body.Close()
	out, err := os.Create(h.StorageFolder + "kubelet.pprof")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error creating file to save result of kubelet profiling for node" + h.NodeIP))
		hlog.Errorf("Error reating file to save result of kubelet profiling for node %s: %v", h.NodeIP, err)
		return
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error saving result of kubelet profiling for node" + h.NodeIP))
		hlog.Errorf("Error saving result of kubelet profiling for node %s: %v", h.NodeIP, err)
		return
	}

	_, err = w.Write([]byte("Successfullly sent profiling request and saved results to "))
	if err != nil {
		hlog.Errorf("could not write response: %v", err)
	}
}
