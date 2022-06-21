package handlers

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

const (
	url = "http://127.0.0.1:6060/debug/pprof/profile"
)

func TestProfileCrio(t *testing.T) {
	testCases := []struct {
		name       string
		client     *http.Client
		storageDir string
		expected   runs.ProfilingRun
	}{
		{
			name: "CrioProfiling passes, ProfileRun returned",
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					// Send response to be tested

					Body: ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			storageDir: "/tmp",
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: true,
				Error:      "",
			},
		},
		{
			name: "Network error, ProfilingRun contains error",
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			storageDir: "/tmp",
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: false,
				Error:      fmt.Sprintf("error with HTTP request for crio profiling %s: statusCode %d", url, http.StatusBadRequest),
			},
		},
		{
			name: "IO error at storing result, ProfilingRun contains error",
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			storageDir: "/inexistingFolder",
			expected: runs.ProfilingRun{
				Type:       runs.CrioRun,
				Successful: false,
				Error:      fmt.Sprintf("error fileHandler - crio profiling for node %s: %s", "127.0.0.1", "/tmp"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandlers("abc", x509.NewCertPool(), tc.storageDir, "127.0.0.1")

			pr := h.profileCrio("1234", tc.client)
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
