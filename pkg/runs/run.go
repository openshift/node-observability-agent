package runs

import (
	"time"

	"github.com/google/uuid"
)

type RunType string

const (
	KubeletRun   RunType = "Kubelet"
	CrioRun      RunType = "CRIO"
	UnknownRun   RunType = "Unknown"
	ScriptingRun RunType = "Scripting"
)

// ExecutionRun holds the status of a CRIO, Kubelet Profiling and scripting execution
type ExecutionRun struct {
	Type       RunType
	Successful bool
	BeginTime  time.Time
	EndTime    time.Time
	Error      string
}

// Run holds the status of a request to the node observability agent
type Run struct {
	ID            uuid.UUID
	ExecutionRuns []ExecutionRun
}
