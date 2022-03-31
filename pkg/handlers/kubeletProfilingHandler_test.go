package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewHTTPTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewHTTPTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestProfileKubelet(t *testing.T) {
	testCases := []struct {
		name     string
		handlers *Handlers
		client   *http.Client
		expected runs.ProfilingRun
	}{
		{
			name:     "KubeletProfiling passes, ProfileRun returned",
			handlers: NewHandlers("abc", "/tmp", "/tmp/fakeSocket", "127.0.0.1"),
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					// Send response to be tested

					Body: ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: true,
				Error:      "",
			},
		},
		{
			name:     "HTTP request 401, ProfileRun in error",
			handlers: NewHandlers("abc", "/tmp", "/tmp/fakeSocket", "127.0.0.1"),

			//client wouldnt be used in this case
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,

					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: statusCode %d", "127.0.0.1", http.StatusUnauthorized),
			},
		},
		{
			name:     "KubeletProfiling fails to save, ProfileRun in error",
			handlers: NewHandlers("abc", "non-existent-path", "/tmp/fakeSocket", "127.0.0.1"),
			client: NewHTTPTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					// Send response to be tested

					Body: ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("error fileHandler - kubelet profiling for node %s: open non-existent-path/kubelet-1234.pprof: no such file or directory", "127.0.0.1"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := tc.handlers.profileKubelet("1234", tc.client)
			if tc.expected.Type != pr.Type {
				t.Errorf("Expecting a ProfilingRun of type %s but was %s", tc.expected.Type, pr.Type)
			}
			if pr.BeginTime.After(pr.EndTime) {
				t.Errorf("Expecting the registered beginDate %v to be before the profiling endDate %v but was not", pr.BeginTime, pr.EndTime)
			}
			if tc.expected.Successful != pr.Successful {
				t.Errorf("Expecting ProfilingRun to be successful=%t but was %t", tc.expected.Successful, pr.Successful)
			}
			if !tc.expected.Successful && !pr.Successful {
				if !strings.Contains(pr.Error, tc.expected.Error) {
					t.Errorf("Error message differs from expected:\nExpected:%s\nGot:%s", tc.expected.Error, pr.Error)
				}
			}
		})
	}
}
