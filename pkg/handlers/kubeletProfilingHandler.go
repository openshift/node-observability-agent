package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func (h *Handlers) ProfileKubelet(uid string) ProfilingRun {
	run := ProfilingRun{
		Type:      KubeletRun,
		BeginDate: time.Now(),
	}
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
		run.EndDate = time.Now()
		run.Error = fmt.Errorf("error preparing http request https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		return run
	}
	//Handle HTTP Req
	res, err := client.Do(req)
	if err != nil {
		run.EndDate = time.Now()
		run.Error = fmt.Errorf("error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		return run
	}

	defer res.Body.Close()
	out, err := os.Create(h.StorageFolder + "kubelet-" + uid + ".pprof")
	if err != nil {
		run.EndDate = time.Now()
		run.Error = fmt.Errorf("error creating file to save result of kubelet profiling for node %s: %w", h.NodeIP, err)
		return run
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	if err != nil {
		run.EndDate = time.Now()
		run.Error = fmt.Errorf("error saving result of kubelet profiling for node %s: %v", h.NodeIP, err)
		return run
	}
	run.EndDate = time.Now()
	run.Sucessful = true
	return run
}
