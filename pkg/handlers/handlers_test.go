package handlers

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/openshift/node-observability-agent/pkg/runs"
	"github.com/openshift/node-observability-agent/pkg/statelocker"
)

const (
	validUID string = "dd37122b-daaf-4d75-9250-c0747e9c5c47"
)

func TestStatus(t *testing.T) {
	testCases := []struct {
		name           string
		isBusy         bool
		isError        bool
		errFileContent string
		expectedCode   int
		expectedBody   string
	}{
		{
			name:         "Service is ready, HTTP 200",
			isBusy:       false,
			isError:      false,
			expectedCode: 200,
			expectedBody: ready,
		},
		{
			name:         "Service is busy, HTTP 409",
			isBusy:       true,
			isError:      false,
			expectedCode: 409,
			expectedBody: validUID + " still running",
		},
		{
			name:           "Service is in error, HTTP 500",
			isBusy:         false,
			isError:        true,
			expectedCode:   500,
			errFileContent: "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedBody:   validUID + " failed",
		},
		{
			name:           "Service is in error, error file unreadable, HTTP 500",
			isBusy:         false,
			isError:        true,
			expectedCode:   500,
			errFileContent: "{\"ID" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}}",
			expectedBody:   "unable to read error file",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://localhost/node-observability-status", nil)
			w := httptest.NewRecorder()
			h := NewHandlers("abc", makeCACertPool(), "/tmp", "/tmp/fakeSocket", "127.0.0.1", true)
			var cur uuid.UUID
			if tc.isBusy {
				c, _, err := h.stateLocker.Lock()
				defer func() {
					err := h.stateLocker.Unlock()
					if err != nil {
						t.Fatal("unable to release turn")
					}
				}()
				if err != nil {
					t.Errorf("Unexpected error : %v", err)
				}
				cur = c
			}
			if tc.isError {
				// prepare an error file
				errorFile := "/tmp/agent.err"
				err := os.WriteFile(errorFile, []byte(tc.errFileContent), 0600)
				if err != nil {
					t.Error(err)
				}
				defer func() {
					if os.Remove(errorFile) != nil {
						t.Error(err)
					}
				}()
			}
			h.Status(w, r)
			resp := w.Result()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status code %d but was %d", tc.expectedCode, resp.StatusCode)
			}

			defer resp.Body.Close()

			bodyContent, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("error reading response body : %v", err)
			}

			if tc.isBusy && !strings.Contains(string(bodyContent), cur.String()) {
				t.Errorf("The UID returned in the HTTP response should be contain uid %v, but was %v", cur, string(bodyContent))
			}
		})
	}
}

func TestSendUID(t *testing.T) {

	testCases := []struct {
		name         string
		expectedCode int
	}{
		{
			name:         "Nominal case, no errors",
			expectedCode: 200,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			uid := uuid.MustParse(validUID)
			err := sendUID(w, uid)
			if err != nil {
				t.Errorf("error calling createAndSendUID : %v", err)
			}
			resp := w.Result()
			defer resp.Body.Close()
			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status code %d but was %d", tc.expectedCode, resp.StatusCode)
			}
			bodyContent, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("error reading response body : %v", err)
			}
			responseRun := runs.Run{}
			err = json.Unmarshal(bodyContent, &responseRun)
			if err != nil {
				t.Errorf("error unmarshalling response body : %v", err)
			}
			if responseRun.ID != uid {
				t.Errorf("The UID returned in the HTTP response should be the same as the one generated by createAndSendUID:\n run.ID=%v, response body contained %v", uid, responseRun.ID)
			}
		})
	}
}

func TestHandleProfilingStateMgmt(t *testing.T) {
	errorFile := "/tmp/agent.err"
	handlingPollInterval := 100 * time.Millisecond
	handlingPollMaxRetry := 5

	testCases := []struct {
		name           string
		serverState    string
		errFileContent string
		expectedCode   int
		expectedState  statelocker.State
		expectedError  bool
	}{
		{
			name:          "Server is ready creates lock, triggers pprof for crio+kubelet and answers 200",
			serverState:   "ready",
			expectedCode:  http.StatusOK,
			expectedState: statelocker.Taken,
			expectedError: false,
		},
		{
			name:           "Server is in error should send 500 immediately",
			serverState:    "error",
			errFileContent: "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedCode:   http.StatusInternalServerError,
			expectedState:  statelocker.InError,
			expectedError:  false,
		},
		{
			name:           "Server is in error, error file unreadable should send 500 immediately",
			serverState:    "error",
			errFileContent: "{\"ID" + validUID + "\",\"ProfilingRuns\":{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedCode:   http.StatusInternalServerError,
			expectedState:  statelocker.InError,
			expectedError:  true,
		},
		{
			name:          "Server is busy should send 409 immediately",
			serverState:   "busy",
			expectedCode:  http.StatusConflict,
			expectedState: statelocker.Taken,
			expectedError: false,
		},
	}
	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			h := NewHandlers("abc", makeCACertPool(), "/tmp", "/tmp/fakeSocket", "127.0.0.1", true)
			r := httptest.NewRequest("GET", "http://localhost/node-observability-status", nil)
			w := httptest.NewRecorder()
			if tc.serverState == "busy" {
				id, state, err := h.stateLocker.Lock()
				if err != nil {
					t.Errorf("failed to lock the state: %v", err)
				}
				t.Logf("state changed: %s, run ID: %s", state, id)
			}
			if tc.serverState == "error" {
				// prepare an error file
				err := os.WriteFile(errorFile, []byte(tc.errFileContent), 0600)
				if err != nil {
					t.Error(err)
				}
				t.Logf("error file updated: %q", tc.errFileContent)
			}

			h.HandleProfiling(w, r)
			resp := w.Result()

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("expected status code %d but was %d", tc.expectedCode, resp.StatusCode)
			}
			uid, s, err := h.stateLocker.LockInfo()
			if tc.expectedState != s {
				t.Errorf("expected state to become %s, but was %s", tc.expectedState, s)
			}

			if tc.expectedError && err == nil {
				t.Error("error was expected but none was found")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tc.expectedState != statelocker.InError {
				if uid == uuid.Nil {
					t.Error("uid was empty when it shouldnt")
				}
			}

			// wait for the end of the profiling to avoid test collisions
			// unless no profiling was run: when taken state was set by the test itself

			if tc.serverState == "busy" {
				return
			}

			for i := 0; i < handlingPollMaxRetry; i++ {
				_, s, _ = h.stateLocker.LockInfo()
				switch s {
				case statelocker.Free:
					t.Logf("profile handling finished")
					return
				case statelocker.InError:
					t.Logf("profile handling finished")
					if err := os.Remove(errorFile); err != nil {
						t.Fatalf("unable to remove file: %v", err)
					}
					t.Logf("error file removed")
					return
				}
				time.Sleep(handlingPollInterval)
			}
			t.Errorf("timed out waiting for the end of profile handling")
		})
	}
}

func TestProcessResults(t *testing.T) {
	h := NewHandlers("abc", makeCACertPool(), "/tmp", "/tmp/fakeSocket", "127.0.0.1", true)

	crioRunOK := runs.ProfilingRun{
		Type:       runs.CrioRun,
		Successful: true,
		BeginTime:  time.Now(),
		EndTime:    time.Now(),
		Error:      "",
	}
	crioRunKO := runs.ProfilingRun{
		Type:       runs.CrioRun,
		Successful: false,
		BeginTime:  time.Now(),
		EndTime:    time.Now(),
		Error:      "fake error",
	}
	kubeletRunOK := runs.ProfilingRun{
		Type:       runs.KubeletRun,
		Successful: true,
		BeginTime:  time.Now(),
		EndTime:    time.Now(),
		Error:      "",
	}
	kubeletRunKO := runs.ProfilingRun{
		Type:       runs.KubeletRun,
		Successful: false,
		BeginTime:  time.Now(),
		EndTime:    time.Now(),
		Error:      "fake error",
	}
	chanAllOK := make(chan runs.ProfilingRun, 2)
	chanAllOK <- kubeletRunOK
	chanAllOK <- crioRunOK

	chanCrioKO := make(chan runs.ProfilingRun, 2)
	chanCrioKO <- kubeletRunOK
	chanCrioKO <- crioRunKO

	chanKubeletKO := make(chan runs.ProfilingRun, 2)
	chanKubeletKO <- kubeletRunKO
	chanKubeletKO <- crioRunOK

	chanOnlyCrio := make(chan runs.ProfilingRun, 1)
	chanOnlyCrio <- crioRunOK

	testCases := []struct {
		name                   string
		channel                chan runs.ProfilingRun
		expectedLock           bool
		expectedError          bool
		expectedTimeout        bool
		expectedCrioSuccess    bool
		expectedKubeletSuccess bool
		expectedRunID          string
	}{
		{
			name:                   "channel with both results OK releases the lock",
			channel:                chanAllOK,
			expectedLock:           false,
			expectedError:          false,
			expectedTimeout:        false,
			expectedCrioSuccess:    true,
			expectedKubeletSuccess: true,
			expectedRunID:          validUID,
		},
		{
			name:                   "channel with crio result KO releases the lock and creates error file",
			channel:                chanCrioKO,
			expectedLock:           false,
			expectedError:          true,
			expectedTimeout:        false,
			expectedCrioSuccess:    false,
			expectedKubeletSuccess: true,
			expectedRunID:          validUID,
		},
		{
			name:                   "channel with kubelet result KO releases the lock and creates error file",
			channel:                chanKubeletKO,
			expectedLock:           false,
			expectedError:          true,
			expectedTimeout:        false,
			expectedCrioSuccess:    true,
			expectedKubeletSuccess: false,
			expectedRunID:          validUID,
		},
		{
			name:                   "channel with only crio result should be unstuck after 40s with error file",
			channel:                chanOnlyCrio,
			expectedLock:           false,
			expectedError:          true,
			expectedTimeout:        true,
			expectedCrioSuccess:    true,
			expectedKubeletSuccess: false,
			expectedRunID:          validUID,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			_, _, err := h.stateLocker.Lock()
			defer func() {
				err := h.stateLocker.Unlock()
				if err != nil {
					t.Fatal("unable to release turn")
				}
			}()
			if err != nil {
				t.Errorf("Unexpected error : %v", err)
			}
			defer cleanup(t)
			h.processResults(uuid.MustParse(validUID), tc.channel)
			uid, s, err := h.stateLocker.LockInfo()
			if err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if !tc.expectedLock && s == statelocker.Taken {
				t.Errorf("Shouldnt be locked but was locked by %v", uid)
			}
			if tc.expectedLock && s == statelocker.Free {
				t.Errorf("Should be locked but wasnt")
			}
			if !tc.expectedError {
				_, err := os.Stat("/tmp/" + validUID + ".log")
				if err != nil {
					t.Errorf("Expected log file /tmp/%s.log but file wasnt there", validUID)
				}
				theRun, err := readRunFromFile("/tmp/" + validUID + ".log")
				if err != nil {
					t.Errorf("error reading file /tmp/%s.log: %v", validUID, err)
				}
				if theRun.ID.String() != tc.expectedRunID {
					t.Errorf("Expected log file /tmp/%s.log to contain run ID %s but was %s", validUID, theRun.ID.String(), tc.expectedRunID)
				}
				for _, aProfilingRun := range theRun.ProfilingRuns {
					if aProfilingRun.Type == runs.CrioRun {
						if aProfilingRun.Successful != tc.expectedCrioSuccess {
							t.Errorf("Expected log file /tmp/%s.log to contain crio run success = %t, but was %t", validUID, tc.expectedCrioSuccess, aProfilingRun.Successful)
						}
					}
					if aProfilingRun.Type == runs.KubeletRun {
						if aProfilingRun.Successful != tc.expectedKubeletSuccess {
							t.Errorf("Expected log file /tmp/%s.log to contain kubelet run success = %t, but was %t", validUID, tc.expectedKubeletSuccess, aProfilingRun.Successful)
						}
					}
				}
			} else {
				_, err := os.Stat("/tmp/agent.err")
				if err != nil {
					t.Errorf("Expected error file /tmp/agent.err but file wasnt there")
				}
				theRun, err := readRunFromFile("/tmp/agent.err")
				if err == nil {
					if theRun.ID.String() != tc.expectedRunID {
						t.Errorf("Expected log file /tmp/agent.err to contain run ID %s but was %s", theRun.ID.String(), tc.expectedRunID)
					}
					for _, aProfilingRun := range theRun.ProfilingRuns {
						if aProfilingRun.Type == runs.CrioRun {
							if aProfilingRun.Successful != tc.expectedCrioSuccess {
								t.Errorf("Expected log file /tmp/agent.err to contain crio run success = %t, but was %t", tc.expectedCrioSuccess, aProfilingRun.Successful)
							}
						}
						if aProfilingRun.Type == runs.KubeletRun {
							if aProfilingRun.Successful != tc.expectedKubeletSuccess {
								t.Errorf("Expected log file /tmp/agent.err to contain kubelet run success = %t, but was %t", tc.expectedKubeletSuccess, aProfilingRun.Successful)
							}
						}
						if aProfilingRun.Type == runs.UnknownRun && !tc.expectedTimeout {
							t.Error("timeout when none was expected")
						}
					}
				}
			}
		})
	}
}

func TestOutputFilePaths(t *testing.T) {
	h := Handlers{StorageFolder: "/fakedir"}

	if expected, got := "/fakedir/crio-fakeid.pprof", h.crioPprofOutputFilePath("fakeid"); expected != got {
		t.Errorf("Wrong crio output path, expected: %q, but got: %q", expected, got)
	}

	if expected, got := "/fakedir/kubelet-fakeid.pprof", h.kubeletPprofOutputFilePath("fakeid"); expected != got {
		t.Errorf("Wrong kubelet output path, expected: %q, but got: %q", expected, got)
	}

	if expected, got := fmt.Sprintf("/fakedir/%s.log", validUID), h.runLogOutputFilePath(runs.Run{ID: uuid.MustParse(validUID)}); expected != got {
		t.Errorf("Wrong log output path, expected: %q, but got: %q", expected, got)
	}

	if expected, got := "/fakedir/agent.err", h.errorOutputFilePath(); expected != got {
		t.Errorf("Wrong log output path, expected: %q, but got: %q", expected, got)
	}
}

func cleanup(t *testing.T) {
	_, err := os.Stat("/tmp/agent.err")
	if err == nil {
		if err := os.Remove("/tmp/agent.err"); err != nil {
			t.Error(err)
		}
	}
	_, err = os.Stat("/tmp/" + validUID + ".log")
	if err == nil {
		if err := os.Remove("/tmp/" + validUID + ".log"); err != nil {
			t.Error(err)
		}
	}
}

func readRunFromFile(fileName string) (runs.Run, error) {
	var arun *runs.Run = &runs.Run{}
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return *arun, err
	}
	err = json.Unmarshal(contents, arun)
	if err != nil {
		return *arun, err
	}
	return *arun, nil
}

func makeCACertPool() *x509.CertPool {
	content, err := os.ReadFile("../../test_resources/kubelet-serving-ca.crt")
	if err != nil {
		panic("Unable to load CACerts file")
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(content) {
		panic("Unable to load CACerts file into CertPool")

	}
	return caCertPool
}
