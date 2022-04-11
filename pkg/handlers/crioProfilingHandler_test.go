package handlers

import (
	"fmt"
	"testing"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
)

func TestProfileCrio(t *testing.T) {
	testCases := []struct {
		name      string
		connector connectors.CmdWrapper
		expected  runs.ProfilingRun
	}{
		{
			name:      "Curl command successful, OK",
			connector: &connectors.FakeConnector{Flag: connectors.NoError},
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: true,
				Error:      "",
			},
		},
		{
			name:      "Network error on curl, ProfilingRun contains error",
			connector: &connectors.FakeConnector{Flag: connectors.SocketErr},
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: false,
				Error:      fmt.Sprintf("error running CRIO profiling :\n%s", "curl: (7) Couldn't connect to server"),
			},
		},
		{
			name:      "IO error at storing result, ProfilingRun contains error",
			connector: &connectors.FakeConnector{Flag: connectors.WriteErr},
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: false,
				Error:      fmt.Sprintf("error running CRIO profiling :\n%s", "curl: (23) Failure writing output to destination"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandlers("abc", []byte("fakeCert"), "/tmp", "/tmp/fakeSocket", "127.0.0.1")
			tc.connector.Prepare("curl", []string{"--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", h.StorageFolder + "crio-1234.pprof"})

			pr := h.profileCrio("1234", tc.connector)
			if tc.expected.Type != pr.Type {
				t.Errorf("Expecting a ProfilingRun of type %s but was %s", tc.expected.Type, pr.Type)
			}
			if pr.BeginTime.After(pr.EndTime) {
				t.Errorf("Expecting the registered beginDate %v to be before the profiling endDate %v but was not", pr.BeginTime, pr.EndTime)
			}
			if tc.expected.Successful != pr.Successful {
				t.Errorf("Expecting ProfilingRun to be successful=%t but was %t", tc.expected.Successful, pr.Successful)
			}
		})
	}

}
