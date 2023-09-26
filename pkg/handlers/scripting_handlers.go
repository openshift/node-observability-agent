package handlers

import (
	"fmt"
	"time"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
)

// executeScript executes the embedded script on the h.NodeIP
func (h *Handlers) executeScript(uid string, cmd connectors.CmdWrapper) runs.ExecutionRun {
	run := runs.ExecutionRun{
		Type:      runs.ScriptingRun,
		BeginTime: time.Now(),
	}

	message, err := cmd.CmdExec()
	run.EndTime = time.Now()
	if err != nil {
		run.Error = fmt.Sprintf("error executing script :\n%s", message)
	} else {
		run.Successful = true
		hlog.Infof("%s", message)
	}
	return run
}
