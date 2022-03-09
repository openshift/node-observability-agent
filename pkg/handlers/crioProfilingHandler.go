package handlers

import (
	"fmt"
	"time"

	"github.com/openshift/node-observability-agent/pkg/connectors"
)

// ProfileCrio calls /debug/pprof/profile on the h.NodeIP, through the unix socket,
// thus triggering a CRIO profiling on that node.
// This call requires access to the host socket, which is passed to the agent in parameter crioSocket
func (h *Handlers) profileCrio(uid string, cmd connectors.CmdWrapper) profilingRun {
	//curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile > /mnt/prof.out

	run := profilingRun{
		Type:      crioRun,
		BeginDate: time.Now(),
	}

	message, err := cmd.CmdExec()
	run.EndDate = time.Now()
	if err != nil {
		run.Error = fmt.Sprintf("error running CRIO profiling :\n%s", message)
	} else {
		run.Successful = true
		hlog.Infof("CRIO profiling successful, %s", message)
	}
	return run
}
