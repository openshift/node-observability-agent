package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var hlog = logrus.WithField("module", "handler")

type Handlers struct {
	Token          string
	NodeIP         string
	StorageFolder  string
	CrioUnixSocket string
}

type RunType string
type FileType string

const (
	KubeletRun RunType  = "Kubelet"
	CRIORun    RunType  = "CRIO"
	LockFile   FileType = "lock"
	LogFile    FileType = "log"
	ErrorFile  FileType = "err"
)

type ProfilingRun struct {
	Type      RunType
	Sucessful bool
	BeginDate time.Time
	EndDate   time.Time
	Error     error
}

type Run struct {
	ID            uuid.UUID
	ProfilingRuns []ProfilingRun
}

func NewHandlers(token string, storageFolder string, crioUnixSocket string, nodeIP string) *Handlers {

	return &Handlers{
		Token:          token,
		NodeIP:         nodeIP,
		StorageFolder:  storageFolder,
		CrioUnixSocket: crioUnixSocket,
	}
}

func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("OK"))
	if err != nil {
		hlog.Errorf("could not write response: %v", err)
	}
}

func createAndSendUID(w http.ResponseWriter, r *http.Request) (Run, error) {

	id := uuid.New()
	response := Run{
		ID: id,
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Server does not support Flusher!",
			http.StatusInternalServerError)
		return response, fmt.Errorf("no support for Flusher")
	}

	jsResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return response, err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsResponse)
	if err != nil {
		hlog.Errorf("Unable to send HTTP response : %v", err)
	}
	flusher.Flush()
	return response, nil
}

func writeRunToFile(run Run, storageFolder string, fileType FileType) string {

	fileName := storageFolder + run.ID.String() + "." + string(fileType)
	bytes, err := json.Marshal(run)
	if err != nil {
		panic("error while creating " + string(fileType) + " file : unable to marshal run of ID" + run.ID.String() + "\n" + err.Error())
	}
	err = os.WriteFile(fileName, bytes, 0644)
	if err != nil {
		panic("error creating " + string(fileType) + "file" + err.Error())
	}
	return fileName
}

func (h *Handlers) HandleProfiling(w http.ResponseWriter, r *http.Request) {
	// Channel for collecting results of profiling
	runResultsChan := make(chan ProfilingRun)

	// Send a HTTP 200 straight away
	run, err := createAndSendUID(w, r)
	if err != nil {
		hlog.Error(err)
		return
	}

	// Create a lock file with a begin date and a uid
	lockFile := writeRunToFile(run, h.StorageFolder, LockFile)

	// Launch both profilings in parallel
	go func() {
		runResultsChan <- h.ProfileKubelet(run.ID.String())
	}()

	go func() {
		runResultsChan <- h.ProfileCrio(run.ID.String())
	}()

	// wait for the results
	run.ProfilingRuns = []ProfilingRun{<-runResultsChan, <-runResultsChan}

	// Process the results
	var errorMessage bytes.Buffer
	var logMessage bytes.Buffer
	for _, aRun := range run.ProfilingRuns {
		if aRun.Error != nil {
			errorMessage.WriteString("errors encountered while running " + string(aRun.Type) + ":\n")
			errorMessage.WriteString(aRun.Error.Error() + "\n")
		}
		logMessage.WriteString(string(aRun.Type) + " - " + run.ID.String() + ": " + aRun.BeginDate.String() + " -> " + aRun.EndDate.String() + "\n")
	}

	// replace the lock file by error file in case of errors
	if errorMessage.Len() > 0 {
		hlog.Error(errorMessage.String())
		err = os.Rename(lockFile, h.StorageFolder+run.ID.String()+string(ErrorFile))
		if err != nil {
			hlog.Errorf("Unable to rename lock file into error file for run %s: %v", run.ID.String(), err)
		}
		writeRunToFile(run, h.StorageFolder, ErrorFile)
		return
	}

	// no errors : simply log the results and rename lock to log
	hlog.Info(logMessage.String())
	err = os.Rename(lockFile, h.StorageFolder+run.ID.String()+"."+string(LogFile))
	if err != nil {
		hlog.Errorf("Unable to rename lock file into log file for run %s: %v", run.ID.String(), err)
	}
	writeRunToFile(run, h.StorageFolder, LogFile)
}
