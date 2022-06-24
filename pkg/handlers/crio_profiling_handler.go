package handlers

import (
	"fmt"
	"time"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
)

// ProfileCrio calls /debug/pprof/profile on the h.NodeIP, through the unix socket,
// thus triggering a CRIO profiling on that node.
// This call requires access to the host socket, which is passed to the agent in parameter crioSocket
func (h *Handlers) profileCrio(uid string, cmd connectors.CmdWrapper) runs.ProfilingRun {
	//curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile > /mnt/prof.out

	run := runs.ProfilingRun{
		Type:      runs.CrioRun,
		BeginTime: time.Now(),
	}

	message, err := cmd.CmdExec()
	run.EndTime = time.Now()
	if err != nil {
		run.Error = fmt.Sprintf("error running CRIO profiling :\n%s", message)
	} else {
		run.Successful = true
		hlog.Infof("CRIO profiling successful, %s", message)
	}
	return run
}
