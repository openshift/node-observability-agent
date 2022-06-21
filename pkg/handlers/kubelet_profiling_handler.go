package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

// ProfileKubelet calls /debug/pprof/profile on the h.NodeIP, thus triggering a kubelet
// profiling on that node.
// This call requires an Authorization header, to which the h.Token is passed as Bearer token
func (h *Handlers) profileKubelet(uid string, client *http.Client) runs.ProfilingRun {
	run := runs.ProfilingRun{
		Type:      runs.KubeletRun,
		BeginTime: time.Now(),
	}

	//Construct HTTP Req
	req, err := http.NewRequest("GET", "https://"+h.NodeIP+":10250/debug/pprof/profile", nil)
	req.Header.Add("Authorization", "Bearer "+h.Token)
	if err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error preparing http request https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		return run
	}
	//Handle HTTP Req
	res, err := client.Do(req)
	if err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: %v", h.NodeIP, err)
		return run
	}

	if res.StatusCode != http.StatusOK {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: statusCode %d", h.NodeIP, res.StatusCode)
		return run
	}

	defer res.Body.Close()
	errFile := h.fileHandler(uid, "kubelet", &res.Body)
	if errFile != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error fileHandler - kubelet profiling for node %s: %s", h.NodeIP, errFile)
		return run
	}

	run.EndTime = time.Now()
	run.Successful = true
	return run
}
