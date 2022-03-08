package handlers

import (
	"fmt"
	"testing"

	"github.com/openshift/node-observability-agent/pkg/connectors"
)

func TestProfileCrio(t *testing.T) {
	testCases := []struct {
		name      string
		connector connectors.CmdWrapper
		expected  ProfilingRun
	}{
		{
			name:      "Curl command successful, OK",
			connector: &connectors.MockConnector{Flag: connectors.NO_ERROR},
			expected: ProfilingRun{
				Type:       CRIORun,
				Successful: true,
				Error:      "",
			},
		},
		{
			name:      "Network error on curl, ProfilingRun contains error",
			connector: &connectors.MockConnector{Flag: connectors.SOCKET_ERR},
			expected: ProfilingRun{
				Type:       CRIORun,
				Successful: false,
				Error:      fmt.Sprintf("error running CRIO profiling :\n%s", "curl: (7) Couldn't connect to server"),
			},
		},
		{
			name:      "IO error at storing result, ProfilingRun contains error",
			connector: &connectors.MockConnector{Flag: connectors.WRITE_ERR},
			expected: ProfilingRun{
				Type:       CRIORun,
				Successful: false,
				Error:      fmt.Sprintf("error running CRIO profiling :\n%s", "curl: (23) Failure writing output to destination"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandlers("abc", "/tmp", "/tmp/fakeSocket", "127.0.0.1")
			tc.connector.Prepare("curl", []string{"--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", h.StorageFolder + "crio-1234.pprof"})

			pr := h.ProfileCrio("1234", tc.connector)
			if tc.expected.Type != pr.Type {
				t.Errorf("Expecting a ProfilingRun of type %s but was %s", tc.expected.Type, pr.Type)
			}
			if pr.BeginDate.After(pr.EndDate) {
				t.Errorf("Expecting the registered beginDate %v to be before the profiling endDate %v but was not", pr.BeginDate, pr.EndDate)
			}
			if tc.expected.Successful != pr.Successful {
				t.Errorf("Expecting ProfilingRun to be successful=%t but was %t", tc.expected.Successful, pr.Successful)
			}
		})
	}

}
