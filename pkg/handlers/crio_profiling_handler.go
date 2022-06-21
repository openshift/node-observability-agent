package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

// ProfileCrio calls /debug/pprof/profile on the h.NodeIP
// crio profiling over http needs to be enabled before attempting
// This call requires access to the host network namespace
func (h *Handlers) profileCrio(uid string, client *http.Client) runs.ProfilingRun {
	url := "http://127.0.0.1:6060/debug/pprof/profile"
	run := runs.ProfilingRun{
		Type:      runs.CrioRun,
		BeginTime: time.Now(),
	}

	resp, err := client.Get(url)
	if err != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error with HTTP request for crio profiling %s: %v", url, err)
		return run
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error with HTTP request for crio profiling %s: statusCode %d", url, resp.StatusCode)
		return run
	}

	errFile := h.fileHandler(uid, "crio", &resp.Body)
	if errFile != nil {
		run.EndTime = time.Now()
		run.Error = fmt.Sprintf("error fileHandler - crio profiling for node %s: %s", "127.0.0.1", errFile)
		return run
	}

	run.EndTime = time.Now()
	run.Successful = true

	return run
}
