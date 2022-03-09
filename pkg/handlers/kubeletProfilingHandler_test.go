package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewHttpTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestProfileKubelet(t *testing.T) {
	testCases := []struct {
		name     string
		handlers *Handlers
		client   *http.Client
		expected ProfilingRun
	}{
		{
			name:     "KubeletProfiling passes, ProfileRun returned",
			handlers: NewHandlers("abc", "/tmp", "/tmp/fakeSocket", "127.0.0.1"),
			client: NewHttpTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					// Send response to be tested

					Body: ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: ProfilingRun{
				Type:       KubeletRun,
				Successful: true,
				Error:      "",
			},
		},
		{
			name:     "HTTP request 401, ProfileRun in error",
			handlers: NewHandlers("abc", "/tmp", "/tmp/fakeSocket", "127.0.0.1"),

			//client wouldnt be used in this case
			client: NewHttpTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,

					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: ProfilingRun{
				Type:       KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("error with HTTP request for kubelet profiling https://%s:10250/debug/pprof/profile: statusCode %d", "127.0.0.1", http.StatusUnauthorized),
			},
		},
		{
			name:     "KubeletProfiling fails to save, ProfileRun in error",
			handlers: NewHandlers("abc", "C:\\", "/tmp/fakeSocket", "127.0.0.1"),
			client: NewHttpTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					// Send response to be tested

					Body: ioutil.NopCloser(bytes.NewBuffer([]byte("OK"))),
					// Must be set to non-nil value or it panics
					Header: make(http.Header),
				}
			}),
			expected: ProfilingRun{
				Type:       KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("error creating file to save result of kubelet profiling for node %s", "127.0.0.1"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := tc.handlers.ProfileKubelet("1234", tc.client)
			if tc.expected.Type != pr.Type {
				t.Errorf("Expecting a ProfilingRun of type %s but was %s", tc.expected.Type, pr.Type)
			}
			if pr.BeginDate.After(pr.EndDate) {
				t.Errorf("Expecting the registered beginDate %v to be before the profiling endDate %v but was not", pr.BeginDate, pr.EndDate)
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
