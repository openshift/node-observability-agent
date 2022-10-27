package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

const (
	fakeMethod     = "get"
	fakeURL        = "http://fakehost:8080/debug"
	fakeToken      = ""
	fakeOutputFile = "fakefile"
)

func TestSendHTTPProfileRequest(t *testing.T) {
	testCases := []struct {
		name             string
		client           *http.Client
		expectedRun      runs.ProfilingRun
		expectedContents string
	}{
		{
			name: "Nominal",
			client: newHTTPTestClient(func(req *http.Request) (*http.Response, error) {
				return newTestResponse("OK", nil, http.StatusOK), nil
			}),
			expectedRun: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: true,
				Error:      "",
			},
			expectedContents: "OK",
		},
		{
			name: "HTTP query error",
			client: newHTTPTestClient(func(req *http.Request) (*http.Response, error) {
				return newTestResponse("", nil, http.StatusUnauthorized), fmt.Errorf("fake error")
			}),
			expectedRun: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("failed sending profiling request: %s %q: fake error", cases.Title(language.Und).String(fakeMethod), fakeURL),
			},
		},
		{
			name: "HTTP response status not OK",
			client: newHTTPTestClient(func(req *http.Request) (*http.Response, error) {
				return newTestResponse("", nil, http.StatusUnauthorized), nil
			}),
			expectedRun: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: false,
				Error:      "error status code received: 401",
			},
		},
		{
			name: "Write to file failed",
			client: newHTTPTestClient(func(req *http.Request) (*http.Response, error) {
				return newTestResponse("OK", fmt.Errorf("fake error"), http.StatusOK), nil
			}),
			expectedRun: runs.ProfilingRun{
				Type:       runs.KubeletRun,
				Successful: false,
				Error:      fmt.Sprintf("failed writing profiling data into file: failed to write to file .+/%s: fake error", fakeOutputFile),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			pr := sendHTTPProfileRequest(runs.UnknownRun, strings.ToUpper(fakeMethod), fakeURL, fakeToken, dir+"/"+fakeOutputFile, tc.client)
			if tc.expectedRun.Successful != pr.Successful {
				t.Errorf("Expecting ProfilingRun successful to be %t but got %t", tc.expectedRun.Successful, pr.Successful)
			}
			if matched, err := regexp.Match(tc.expectedRun.Error, []byte(pr.Error)); err != nil {
				t.Errorf("Failed to match ProfilingRun error: %v", err)
			} else if !matched {
				t.Errorf("Expecting ProfilingRun error %q to match %q regexp", pr.Error, tc.expectedRun.Error)
			}
			if pr.BeginTime.After(pr.EndTime) {
				t.Errorf("Expecting begin time %v to be before the end time %v", pr.BeginTime, pr.EndTime)
			}
			if tc.expectedContents != "" {
				if contents, err := os.ReadFile(dir + "/" + fakeOutputFile); err != nil {
					t.Errorf("Failed to read the contents of kubelet pprof data: %v", err)
				} else {
					if tc.expectedContents != string(contents) {
						t.Errorf("Expecting pprof contents: %q, but got %q", tc.expectedContents, string(contents))
					}
				}
			}
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type testReadCloser struct {
	*bytes.Reader
	err error
}

func (r *testReadCloser) WriteTo(w io.Writer) (int64, error) {
	// writeToFile uses io.Copy which in turn uses WriteTo method if it's present.
	// Since we embed bytes.Reader, WriteTo is the one to be hooked up.
	if r.err != nil {
		return 0, r.err
	}
	return r.Reader.WriteTo(w)
}

func (r *testReadCloser) Close() error {
	return nil
}

func newHTTPTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
	}
}

func newTestResponse(body string, err error, statusCode int) *http.Response {
	return &http.Response{
		Body: &testReadCloser{
			Reader: bytes.NewReader([]byte(body)),
			err:    err,
		},
		StatusCode: statusCode,
		Header:     make(http.Header),
	}
}
