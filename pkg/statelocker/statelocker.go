package statelocker

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

type StateLocker interface {
	Lock() (uuid.UUID, State, error)
	SetError(runInError runs.Run) error
	Unlock() error
	LockInfo() (uuid.UUID, State, error)
}

// StateLock struct holds the state of the agent service
// and ensures its update in racy conditions
type StateLock struct {
	mux           *sync.Mutex
	takerID       uuid.UUID
	errorFilePath string
}

// NewStateLock creates a mutex for syncing the agent service state
// the pathToErr parameter is the path to the error file which might
// be created in case a profiling request is in error.
func NewStateLock(pathToErr string) *StateLock {
	return &StateLock{
		mux:           &sync.Mutex{},
		errorFilePath: pathToErr,
		takerID:       uuid.Nil,
	}
}

// StateLock attempts to take the single token available
// The first return param is the UID of the job holding the lock
// According to the value of the second return param (state), this
// uuid is either a newly created UUID, or the one from the ongoing job
// The second return parameter is the State: Free is returned in case of success, Taken is returned in case
// a previous job is still running, InError is returned in case the errorFile exists
// The last parameter returned is the error encountered, if any
func (m *StateLock) Lock() (uuid.UUID, State, error) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.takerID != uuid.Nil {
		return m.takerID, Taken, nil
	}
	if m.errorFileExists() {
		uid, err := m.readUIDFromFile()
		if err != nil {
			return uuid.Nil, InError, err
		}
		return uid, InError, nil
	}
	m.takerID = uuid.New()
	return m.takerID, Free, nil
}

func (m *StateLock) SetError(runInError runs.Run) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if runInError.ID != uuid.Nil {
		return m.writeRunToErrorFile(runInError)
	}
	return nil
}

func (m *StateLock) Unlock() error {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.takerID = uuid.Nil
	return nil
}

func (m *StateLock) LockInfo() (uuid.UUID, State, error) {
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

func (m *StateLock) errorFileExists() bool {
	if _, err := os.Stat(m.errorFilePath); err != nil {
		return false
	}
	return true
}

func (m *StateLock) readUIDFromFile() (uuid.UUID, error) {
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

func (m *StateLock) writeRunToErrorFile(arun runs.Run) error {
	bytes, err := json.Marshal(arun)
	if err != nil {
		return fmt.Errorf("error while creating %s file : unable to marshal run of ID %s\n%w", string(m.errorFilePath), arun.ID.String(), err)
	}
	err = os.WriteFile(m.errorFilePath, bytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing  %s file: %w", m.errorFilePath, err)
	}
	return nil
}
