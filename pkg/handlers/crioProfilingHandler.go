package handlers

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

func (h *Handlers) ProfileCrio() ProfilingRun {
	//curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile > /mnt/prof.out
	var stderr bytes.Buffer
	run := ProfilingRun{
		Type:      CRIORun,
		BeginDate: time.Now(),
	}

	cmd := exec.Command("curl", "--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", h.StorageFolder+"crio.pprof")
	cmd.Stderr = &stderr
	err := cmd.Run()
	run.EndDate = time.Now()
	errStr := stderr.String()
	if err != nil {
		run.Error = fmt.Errorf("error running CRIO profiling :\n%s", errStr)
	}
	return run
}
