package handlers

import (
	"fmt"
	"time"

	"github.com/openshift/node-observability-agent/pkg/connectors"
)

func (h *Handlers) ProfileCrio(uid string, cmd connectors.CmdWrapper) ProfilingRun {
	//curl --unix-socket /var/run/crio/crio.sock http://localhost/debug/pprof/profile > /mnt/prof.out

	run := ProfilingRun{
		Type:      CRIORun,
		BeginDate: time.Now(),
	}

	message, err := cmd.CmdExec()
	run.EndDate = time.Now()
	if err != nil {
		run.Error = fmt.Errorf("error running CRIO profiling :\n%s", message)
	} else {
		run.Successful = true
		hlog.Infof("CRIO profiling successful, %s", message)
	}
	return run
}
