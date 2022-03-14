package runs

import (
	"time"

	"github.com/google/uuid"
)

type RunType string

const (
	KubeletRun RunType = "Kubelet"
	CrioRun    RunType = "CRIO"
	UnknownRun RunType = "Unknown"
)

// ProfilingRun holds the status of a CRIO or Kubelet Profiling execution
type ProfilingRun struct {
	Type       RunType
	Successful bool
	BeginDate  time.Time
	EndDate    time.Time
	Error      string
}

// Run holds the status of a request to the node observability agent
type Run struct {
	ID            uuid.UUID
	ProfilingRuns []ProfilingRun
}
