package handlers

import (
	"fmt"
	"time"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
)

// Metrics calls the metrics embedded script on the h.NodeIP
func (h *Handlers) metrics(uid string, cmd connectors.CmdWrapper) runs.ExecutionRun {

	run := runs.ExecutionRun{
		Type:      runs.MetricsRun,
		BeginTime: time.Now(),
	}

	message, err := cmd.CmdExec()
	run.EndTime = time.Now()
	if err != nil {
		run.Error = fmt.Sprintf("error running Metrics script :\n%s", message)
	} else {
		run.Successful = true
		hlog.Infof("%s", message)
	}
	return run
}
