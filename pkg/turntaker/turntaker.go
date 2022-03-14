package turntaker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/google/uuid"

	"github.com/openshift/node-observability-agent/pkg/runs"
)

type State string

const (
	Free    State = "FREE"
	Taken   State = "TAKEN"
	InError State = "ERROR"
)

type SingleTaker interface {
	TakeTurn() (uuid.UUID, uuid.UUID, State, error)
	ReleaseTurn(runInError runs.Run) error
	WhoseTurn() (uuid.UUID, State, error)
}

// TurnTaker struct holds the state of the agent service
// and ensures its update in racy conditions
type TurnTaker struct {
	mux           *sync.RWMutex
	takerID       uuid.UUID
	errorFilePath string
}

// NewTurnTaker creates a mutex for syncing the agent service state
// the pathToErr parameter is the path to the error file which might
// be created in case a profiling request is in error.
func NewTurnTaker(pathToErr string) *TurnTaker {
	return &TurnTaker{
		mux:           &sync.RWMutex{},
		errorFilePath: pathToErr,
		takerID:       uuid.Nil,
	}
}

// TakeTurn attempts to take the single token available
// The first return param is the UID created when success
// The second return parameter is the UID of the job that is currently running (when state is Taken)
// The third return parameter is the State: Free is returned in case of success, Taken is returned in case
// a previous job is still running, InError is returned in case the errorFile exists
// The last parameter returned is the error encountered, if any
func (m *TurnTaker) TakeTurn() (uuid.UUID, uuid.UUID, State, error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.takerID != uuid.Nil {
		return uuid.Nil, m.takerID, Taken, nil
	}
	if m.errorFileExists() {
		uid, err := m.readUIDFromFile()
		if err != nil {
			return uuid.Nil, uuid.Nil, InError, err
		}
		return uid, uid, InError, nil
	}
	m.takerID = uuid.New()
	return m.takerID, uuid.Nil, Free, nil
}

func (m *TurnTaker) ReleaseTurn(runInError runs.Run) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.takerID = uuid.Nil
	if runInError.ID != uuid.Nil {
		return m.writeRunToErrorFile(runInError)
	}
	return nil
}

func (m *TurnTaker) WhoseTurn() (uuid.UUID, State, error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.errorFileExists() {
		uid, err := m.readUIDFromFile()
		if err != nil {
			return uuid.Nil, InError, err
		}
		return uid, InError, nil
	}
	if m.takerID != uuid.Nil {
		return m.takerID, Taken, nil
	}
	return m.takerID, Free, nil
}

func (m *TurnTaker) errorFileExists() bool {
	if _, err := os.Stat(m.errorFilePath); err != nil {
		return false
	}
	return true
}

func (m *TurnTaker) readUIDFromFile() (uuid.UUID, error) {
	var arun *runs.Run = &runs.Run{}
	contents, err := ioutil.ReadFile(m.errorFilePath)
	if err != nil {
		return uuid.Nil, err
	}
	err = json.Unmarshal(contents, arun)
	if err != nil {
		return uuid.Nil, err
	}
	return arun.ID, nil
}

func (m *TurnTaker) writeRunToErrorFile(arun runs.Run) error {
	bytes, err := json.Marshal(arun)
	if err != nil {
		return fmt.Errorf("error while creating %s file : unable to marshal run of ID %s\n%v", string(m.errorFilePath), arun.ID.String(), err)
	}
	err = os.WriteFile(m.errorFilePath, bytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing  %s file: %v", m.errorFilePath, err)
	}
	return nil
}
