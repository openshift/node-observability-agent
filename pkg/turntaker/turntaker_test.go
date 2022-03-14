package turntaker

import (
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

const (
	validUID string = "dd37122b-daaf-4d75-9250-c0747e9c5c47"
)

func TestTakeTurn(t *testing.T) {
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
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			var previousJobId uuid.UUID
			turnTaker := NewTurnTaker("/tmp/agent.err")
			if tC.previousState == InError {
				previousJobId = uuid.MustParse(validUID)
				// prepare an error file
				errorFile := "/tmp/agent.err"
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
				id, _, _, err := turnTaker.TakeTurn()
				defer func() {
					err := turnTaker.ReleaseTurn(runs.Run{})
					if err != nil {
						t.Fatal("unable to release turn")
					}
				}()
				if err != nil {
					t.Fatalf("Unexpected error preparing test: %v", err)
				}
				previousJobId = id
			}

			curID, prevID, s, err := turnTaker.TakeTurn()
			defer func() {
				err := turnTaker.ReleaseTurn(runs.Run{})
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
			if tC.previousState == Taken {
				if previousJobId != prevID {
					t.Errorf("expected turn to be with %v, but was %v", previousJobId, prevID)
				}
			}
			if tC.previousState == Free {
				if curID == uuid.Nil {
					t.Error("expected id of new job to be returned but was empty")
				}
			}
			if tC.previousState == InError && !tC.expectedError {
				if previousJobId != prevID {
					t.Errorf("expected turn to be with %v, but was %v", previousJobId, prevID)
				}
			}

		})
	}
}

func TestErrorFileExists(t *testing.T) {
	tt := NewTurnTaker("/tmp/agent.err")
	testCases := []struct {
		name      string
		turnTaker TurnTaker
		expected  bool
	}{
		{
			name:      "file doesnt exist, return false",
			turnTaker: *tt,
			expected:  false,
		},
		{
			name:      "file exists, return true",
			turnTaker: *tt,
			expected:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expected {
				// prepare an error file
				errorFile := "/tmp/agent.err"
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
			result := tc.turnTaker.errorFileExists()
			if tc.expected != result {
				t.Errorf("Expected errorFileExists = %v but was %v", tc.expected, result)
			}
		})
	}
}
