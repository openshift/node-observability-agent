package statelocker

import (
	"fmt"
	"os"
	"testing"

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
			name:          "turnTaker was free, can take turn",
			previousState: Free,
			expectedState: Free,
			expectedError: false,
		},
		{
			name:          "turnTaker was taken, returns the id of running job",
			previousState: Taken,
			expectedState: Taken,
			expectedError: false,
		},
		{
			name:           "turnTaker was in error, returns the id of job in error",
			previousState:  InError,
			expectedState:  InError,
			errFileContent: "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginDate\":\"2022-03-03T10:10:17.188097819Z\",\"EndDate\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginDate\":\"2022-03-03T10:10:17.188499431Z\",\"EndDate\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedError:  false,
		},
		{
			name:           "turnTaker was in error, and unable to read error file, error",
			previousState:  InError,
			expectedState:  InError,
			errFileContent: "{\"ID" + validUID + "\",\"ProfilingRuns\":{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginDate\":\"2022-03-03T10:10:17.188097819Z\",\"EndDate\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginDate\":\"2022-03-03T10:10:17.188499431Z\",\"EndDate\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}",
			expectedError:  true,
		},
	}
	for i, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			var previousJobID uuid.UUID
			errFileName := "/tmp/agent_tt" + fmt.Sprint(i) + ".err"
			turnTaker := NewStateLock(errFileName)
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
				id, _, _, err := turnTaker.Lock()
				defer func() {
					err := turnTaker.Unlock(runs.Run{})
					if err != nil {
						t.Fatal("unable to release turn")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
				previousJobID = id
			}

			curID, prevID, s, err := turnTaker.Lock()
			defer func() {
				err := turnTaker.Unlock(runs.Run{})
				if err != nil {
					t.Fatal("unable to release turn")
				}
			}()
			if tC.expectedError && err == nil {
				t.Error("expected error but there were none")
			}
			if !tC.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tC.expectedState != s {
				t.Errorf("expected state to be %s but was %s", tC.expectedState, s)
			}
			if tC.previousState == Taken && previousJobID != prevID {
				t.Errorf("expected turn to be with %v, but was %v", previousJobID, prevID)
			}
			if tC.previousState == Free && curID == uuid.Nil {
				t.Error("expected id of new job to be returned but was empty")
			}
			if tC.previousState == InError && !tC.expectedError && previousJobID != prevID {
				t.Errorf("expected turn to be with %v, but was %v", previousJobID, prevID)
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
			name:          "Turn was Free, Unlock succeeds",
			previousState: Free,
			runInError:    runs.Run{},
			expectedError: false,
			expectedUID:   uuid.Nil,
		},
		{
			name:          "Turn was busy, Unlock succeeds",
			previousState: Taken,
			runInError:    runs.Run{},
			expectedError: false,
			expectedUID:   uuid.Nil,
		},
		{
			name:          "Turn is released with error, Unlock succeeds",
			previousState: Free,
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
			errFileName := "/tmp/agent_rt" + fmt.Sprint(i) + ".err"
			turnTaker := NewStateLock(errFileName)
			if tc.previousState == InError {
				// prepare an error file
				fileContent := "{\"ID\":\"1234\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginDate\":\"2022-03-03T10:10:17.188097819Z\",\"EndDate\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginDate\":\"2022-03-03T10:10:17.188499431Z\",\"EndDate\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
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
				_, _, _, err := turnTaker.Lock()
				defer func() {
					err := turnTaker.Unlock(runs.Run{})
					if err != nil {
						t.Fatal("unable to release turn")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
			}
			err := turnTaker.Unlock(tc.runInError)
			if tc.expectedError && err == nil {
				t.Error("expected error but there were none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("unexpected error : %v", err)
			}
			if tc.expectedUID != turnTaker.takerID {
				t.Errorf("Expected UID to be nil but was %v", turnTaker.takerID)
			}
		})
	}
}

func TestWhoseTurn(t *testing.T) {
	testCases := []struct {
		name          string
		previousState State
		runInError    runs.Run
		expectedError bool
		expectedState State
	}{
		{
			name:          "Turn was Free, whoseTurn returns Free",
			previousState: Free,
			runInError:    runs.Run{},
			expectedError: false,
			expectedState: Free,
		},
		{
			name:          "Turn was Taken, whoseTurn returns Taken",
			previousState: Taken,
			runInError:    runs.Run{},
			expectedError: false,
			expectedState: Taken,
		},
		{
			name:          "Turn was InError, whoseTurn returns InError",
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
				fileContent := "{\"ID\":\"" + validUID + "\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginDate\":\"2022-03-03T10:10:17.188097819Z\",\"EndDate\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginDate\":\"2022-03-03T10:10:17.188499431Z\",\"EndDate\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
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
			turnTaker := NewStateLock(errFileName)
			if tc.previousState == Taken {
				id, _, _, err := turnTaker.Lock()
				expectedID = id
				defer func() {
					err := turnTaker.Unlock(runs.Run{})
					if err != nil {
						t.Fatal("unable to release turn")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
			}
			curID, s, err := turnTaker.LockInfo()
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
				fileContent := "{\"ID\":\"1234\",\"ProfilingRuns\":[{\"Type\":\"Kubelet\",\"Successful\":false,\"BeginDate\":\"2022-03-03T10:10:17.188097819Z\",\"EndDate\":\"2022-03-03T10:10:47.211572681Z\",\"Error\":\"fake error\"},{\"Type\":\"CRIO\",\"Successful\":true,\"BeginDate\":\"2022-03-03T10:10:17.188499431Z\",\"EndDate\":\"2022-03-03T10:10:47.215840909Z\",\"Error\":null}]}"
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
			turnTaker := NewStateLock(errorFile)
			t.Logf("%v", tc)
			result := turnTaker.errorFileExists()
			if tc.expected != result {
				t.Errorf("Expected errorFileExists = %v but was %v", tc.expected, result)
			}
		})
	}
}
