package handlers

import (
	"bytes"
	"net/http"
	"os/exec"
)

func (h *Handlers) ProfileCrio(w http.ResponseWriter, r *http.Request) {
	//curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile > /mnt/prof.out
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("curl", "--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", h.StorageFolder+"crio.pprof")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := stdout.String(), stderr.String()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error running CRIO profiling :\n" + errStr))
		if err != nil {
			hlog.Errorf("could not write response: %v", err)
			return
		}
		hlog.Errorf("Error running CRIO profiling :\n%s", errStr)
		return
	}

	_, err = w.Write([]byte("Successfully sent profiling request and saved results to " + h.StorageFolder + "\n" + outStr))
	if err != nil {
		hlog.Errorf("could not write response: %v", err)
	}
}
