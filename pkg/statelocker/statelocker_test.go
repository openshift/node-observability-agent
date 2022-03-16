package statelocker

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

const (
	validUID string = "dd37122b-daaf-4d75-9250-c0747e9c5c47"
)

func TestLock(t *testing.T) {
	testCases := []struct {
		name           string
		previousState  State
		expectedState  State
		errFileContent string
		expectedError  bool
	}{
		{
			name:          "stateLock was free, can lock",
			previousState: Free,
			expectedState: Free,
			expectedError: false,
		},
		{
			name:          "stateLock was taken, returns the id of running job",
			previousState: Taken,
			expectedState: Taken,
			expectedError: false,
		},
		{
			name:           "stateLock was in error, returns the id of job in error",
			previousState:  InError,
			expectedState:  InError,
			errFileContent: "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedError:  false,
		},
		{
			name:           "stateLock was in error, and unable to read error file, error",
			previousState:  InError,
			expectedState:  InError,
			errFileContent: "{\"ID" + validUID + "\",\"ProfilingRuns\":{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedError:  true,
		},
	}
	for i, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			var previousJobID uuid.UUID
			errFileName := "/tmp/agent_tt" + fmt.Sprint(i) + ".err"
			stateLock := NewStateLock(errFileName)
			if tC.previousState == InError {
				previousJobID = uuid.MustParse(validUID)
				// prepare an error file
				errorFile := errFileName
				err := os.WriteFile(errorFile, []byte(tC.errFileContent), 0644)
				if err != nil {
					t.Error(err)
				}
				defer func() {
					if os.Remove(errorFile) != nil {
						t.Error(err)
					}
				}()
			}
			if tC.previousState == Taken {
				id, _, err := stateLock.Lock()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
				previousJobID = id
			}

			theID, s, err := stateLock.Lock()
			if tC.expectedError && err == nil {
				t.Error("expected error but there were none")
			}
			if !tC.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tC.expectedState != s {
				t.Errorf("expected state to be %s but was %s", tC.expectedState, s)
			}
			if tC.previousState == Taken && previousJobID != theID {
				t.Errorf("expected to be locked by %v, but was %v", previousJobID, theID)
			}
			if tC.previousState == Free && theID == uuid.Nil {
				t.Error("expected id of new job to be returned but was empty")
			}
			if tC.previousState == InError && !tC.expectedError && previousJobID != theID {
				t.Errorf("expected to be locked by %v, but was %v", previousJobID, theID)
			}

		})
	}
}

func TestSetError(t *testing.T) {

	testCases := []struct {
		name          string
		previousState State
		runInError    runs.Run
		expectedError bool
		expectedUID   uuid.UUID
	}{
		{
			name:          "Locked, succeed to write error file",
			previousState: Taken,
			runInError: runs.Run{
				ID: uuid.MustParse(validUID),
				ProfilingRuns: []runs.ProfilingRun{
					{
						Type:       runs.KubeletRun,
						Successful: false,
						BeginTime:  time.Now(),
						EndTime:    time.Now(),
						Error:      "Fake Error",
					},
					{
						Type:       runs.CrioRun,
						Successful: true,
						BeginTime:  time.Now(),
						EndTime:    time.Now(),
						Error:      "",
					},
				},
			},
			expectedError: false,
			expectedUID:   uuid.MustParse(validUID),
		},
	}
	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errFileName := "/tmp/agent_se" + fmt.Sprint(i) + ".err"
			defer func() {
				if err := os.Remove(errFileName); err != nil {
					t.Error(err)
				}
			}()
			stateLock := NewStateLock(errFileName)
			err := stateLock.SetError(tc.runInError)
			if tc.expectedError && err == nil {
				t.Error("expected error but found none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			uid, err := stateLock.readUIDFromFile()
			if tc.expectedError && err == nil {
				t.Error("expected error but found none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if !tc.expectedError && tc.expectedUID != uid {
				t.Errorf("expected uid %v, but was %v", tc.expectedUID, uid)
			}
		})
	}
}
func TestUnlock(t *testing.T) {

	testCases := []struct {
		name          string
		previousState State
		runInError    runs.Run
		expectedError bool
		expectedUID   uuid.UUID
	}{
		{
			name:          "StateLock was Free, Unlock succeeds",
			previousState: Free,
			runInError:    runs.Run{},
			expectedError: false,
			expectedUID:   uuid.Nil,
		},
		{
			name:          "StateLock was busy, Unlock succeeds",
			previousState: Taken,
			runInError:    runs.Run{},
			expectedError: false,
			expectedUID:   uuid.Nil,
		},
		{
			name:          "StateLock was in error, Unlock succeeds",
			previousState: InError,
			runInError: runs.Run{
				ID:            uuid.MustParse(validUID),
				ProfilingRuns: []runs.ProfilingRun{},
			},
			expectedError: false,
			expectedUID:   uuid.Nil,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errFileName := "/tmp/agent_u" + fmt.Sprint(i) + ".err"
			stateLock := NewStateLock(errFileName)
			if tc.previousState == InError {
				// prepare an error file
				fileContent := "{\"ID\":\"1234\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
				err := os.WriteFile(errFileName, []byte(fileContent), 0644)
				if err != nil {
					t.Error(err)
				}
				defer func() {
					if os.Remove(errFileName) != nil {
						t.Error(err)
					}
				}()
			}
			if tc.previousState == Taken {
				_, _, err := stateLock.Lock()
				defer func() {
					err := stateLock.Unlock()
					if err != nil {
						t.Fatal("unable to release lock")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
			}
			err := stateLock.Unlock()
			if tc.expectedError && err == nil {
				t.Error("expected error but there were none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tc.expectedUID != stateLock.takerID {
				t.Errorf("Expected UID to be nil but was %v", stateLock.takerID)
			}
		})
	}
}

func TestLockInfo(t *testing.T) {
	testCases := []struct {
		name          string
		previousState State
		runInError    runs.Run
		expectedError bool
		expectedState State
	}{
		{
			name:          "Lock was Free, LockInfo returns Free",
			previousState: Free,
			runInError:    runs.Run{},
			expectedError: false,
			expectedState: Free,
		},
		{
			name:          "Lock was Taken, LockInfo returns Taken",
			previousState: Taken,
			runInError:    runs.Run{},
			expectedError: false,
			expectedState: Taken,
		},
		{
			name:          "Lock was InError, LockInfo returns InError",
			previousState: InError,
			runInError: runs.Run{
				ID:            uuid.MustParse(validUID),
				ProfilingRuns: []runs.ProfilingRun{},
			},
			expectedError: false,
			expectedState: InError,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errFileName := "/tmp/agent_wt" + fmt.Sprint(i) + ".err"
			expectedID := uuid.Nil
			if tc.previousState == InError {
				expectedID = uuid.MustParse(validUID)
				// prepare an error file
				fileContent := "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
				err := os.WriteFile(errFileName, []byte(fileContent), 0644)
				if err != nil {
					t.Error(err)
				}
				defer func() {
					if os.Remove(errFileName) != nil {
						t.Error(err)
					}
				}()
			}
			stateLock := NewStateLock(errFileName)
			if tc.previousState == Taken {
				id, _, err := stateLock.Lock()
				expectedID = id
				defer func() {
					err := stateLock.Unlock()
					if err != nil {
						t.Fatal("unable to release lock")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
			}
			curID, s, err := stateLock.LockInfo()
			if tc.expectedError && err == nil {
				t.Error("expected error but there were none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tc.expectedState != s {
				t.Errorf("Expected State to be %s but was %v", tc.expectedState, s)
			}
			if expectedID != curID {
				t.Errorf("Expected ID to be %v but was %v", expectedID, curID)
			}
		})
	}
}

func TestErrorFileExists(t *testing.T) {

	testCases := []struct {
		name     string
		expected bool
	}{
		{
			name:     "file doesnt exist, return false",
			expected: false,
		},
		{
			name:     "file exists, return true",
			expected: true,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorFile := "/tmp/agent_efe" + fmt.Sprint(i) + ".err"

			if tc.expected {
				// prepare an error file
				fileContent := "{\"ID\":\"1234\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginTime\":\"2022-03-03T10:10:17.188097819Z\",\"EndTime\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginTime\":\"2022-03-03T10:10:17.188499431Z\",\"EndTime\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
				err := os.WriteFile(errorFile, []byte(fileContent), 0644)
				if err != nil {
					t.Error(err)
				}
				defer func() {
					if os.Remove(errorFile) != nil {
						t.Error(err)
					}
				}()
			}
			stateLock := NewStateLock(errorFile)
			t.Logf("%v", tc)
			result := stateLock.errorFileExists()
			if tc.expected != result {
				t.Errorf("Expected errorFileExists = %v but was %v", tc.expected, result)
			}
		})
	}
}
