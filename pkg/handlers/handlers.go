package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/sirupsen/logrus"
)

var hlog = logrus.WithField("module", "handler")

type Handlers struct {
	Token          string
	NodeIP         string
	StorageFolder  string
	CrioUnixSocket string
	mux            *sync.RWMutex
	onGoingRunId   string
}

type RunType string
type FileType string

const (
	KubeletRun RunType  = "Kubelet"
	CRIORun    RunType  = "CRIO"
	UnknownRun RunType  = "Unknown"
	LockFile   FileType = "lock"
	LogFile    FileType = "log"
	ErrorFile  FileType = "err"
	timeout    int      = 35
)

type ProfilingRun struct {
	Type       RunType
	Successful bool
	BeginDate  time.Time
	EndDate    time.Time
	Error      string
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
		mux:            &sync.RWMutex{},
		onGoingRunId:   "",
	}
}

const (
	ready = "Service is ready"
)

func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	if h.errorFileExists() {
		uid, err := readUidFromFile(filepath.Join(h.StorageFolder, "agent."+string(ErrorFile)))
		if err != nil {
			http.Error(w, "unable to read error file",
				http.StatusInternalServerError)
			hlog.Error("Unable to read error file")
			return
		}
		err = respondBusyOrError(uid, w, r, true)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
		return
	}
	if h.onGoingRunId == "" {
		_, err := w.Write([]byte(ready))
		if err != nil {
			hlog.Errorf("could not send response busy : %v", err)
		}
	} else {
		err := respondBusyOrError(h.onGoingRunId, w, r, false)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
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

func writeRunToFile(run Run, storageFolder string, fileType FileType) (string, error) {
	var fileName string
	if fileType == LogFile {
		fileName = filepath.Join(storageFolder, run.ID.String()+"."+string(fileType))
	} else {
		fileName = filepath.Join(storageFolder, "agent."+string(fileType))
	}

	bytes, err := json.Marshal(run)
	if err != nil {
		return "", fmt.Errorf("error while creating %s file : unable to marshal run of ID %s\n%v", string(fileType), run.ID.String(), err)
	}
	err = os.WriteFile(fileName, bytes, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing  %s file: %v", fileName, err)
	}
	return fileName, nil
}

func (h *Handlers) errorFileExists() bool {
	fileName := filepath.Join(h.StorageFolder, "agent."+string(ErrorFile))
	//TODO return and handle errors better
	if _, err := os.Stat(fileName); err != nil {
		hlog.Errorf("error getting stats for %s:\n%v", fileName, err)
		return false
	} else {
		return true
	}
}

func readUidFromFile(fileName string) (string, error) {
	var run *Run = &Run{}
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(contents, run)
	if err != nil {
		return "", err
	}
	return run.ID.String(), nil
}

func respondBusyOrError(uid string, w http.ResponseWriter, r *http.Request, isError bool) error {

	message := ""
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Server does not support Flusher!",
			http.StatusInternalServerError)
		return fmt.Errorf("no support for Flusher")
	}
	if isError {
		w.WriteHeader(http.StatusInternalServerError)
		message = uid + " failed."
	} else {
		w.WriteHeader(http.StatusConflict)
		message = uid + " still running"
	}
	_, err := w.Write([]byte(message))
	if err != nil {
		hlog.Errorf("Unable to send HTTP response : %v", err)
		return err
	}
	flusher.Flush()
	return nil
}

func (h *Handlers) HandleProfiling(w http.ResponseWriter, r *http.Request) {

	if h.onGoingRunId != "" {
		uid := h.onGoingRunId

		err := respondBusyOrError(uid, w, r, false)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
		return
	} else if h.errorFileExists() {
		uid, err := readUidFromFile(filepath.Join(h.StorageFolder, "agent."+string(ErrorFile)))
		if err != nil {
			http.Error(w, "unable to read error file",
				http.StatusInternalServerError)
			hlog.Error("Unable to read error file")
			return
		}
		err = respondBusyOrError(uid, w, r, true)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
		return
	}

	// Send a HTTP 200 straight away
	run, err := createAndSendUID(w, r)
	if err != nil {
		hlog.Error(err)
		return
	}

	// Create a lock file with a begin date and a uid
	h.mux.Lock()
	h.onGoingRunId = run.ID.String()

	// Channel for collecting results of profiling
	runResultsChan := make(chan ProfilingRun)

	// Launch both profilings in parallel as well as the routine to wait for results
	go func() {
		//TODO Go back to securely making this request
		//Prepare http client that ignores tls check
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: transCfg}
		runResultsChan <- h.ProfileKubelet(run.ID.String(), client)
	}()

	go func() {
		connector := connectors.Connector{}
		connector.Prepare("curl", []string{"--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", filepath.Join(h.StorageFolder, "crio-"+run.ID.String()+".pprof")})
		runResultsChan <- h.ProfileCrio(run.ID.String(), &connector)
	}()

	go func() {
		h.processResults(run, runResultsChan)
	}()
}
func (h *Handlers) processResults(run Run, runResultsChan chan ProfilingRun) {
	// unlock as soon as finished processing
	defer func() {
		h.mux.Unlock()
		h.onGoingRunId = ""
		close(runResultsChan)
	}()
	// wait for the results
	run.ProfilingRuns = []ProfilingRun{}
	isTimeout := false
	for nb := 0; nb < 2 && !isTimeout; {
		select {
		case pr := <-runResultsChan:
			nb++
			run.ProfilingRuns = append(run.ProfilingRuns, pr)
		case <-time.After(time.Second * time.Duration(timeout)):
			//timeout! dont wait anymore
			prInTimeout := ProfilingRun{
				Type:       UnknownRun,
				Successful: false,
				BeginDate:  time.Now(),
				EndDate:    time.Now(),
				Error:      fmt.Sprintf("timeout after waiting %ds", timeout),
			}
			prInTimeout.Error = fmt.Sprintf("timeout after waiting %ds", timeout)
			run.ProfilingRuns = append(run.ProfilingRuns, prInTimeout)
			isTimeout = true
		}
	}

	// Process the results
	var errorMessage bytes.Buffer
	var logMessage bytes.Buffer
	for _, aRun := range run.ProfilingRuns {
		if aRun.Error != "" {
			errorMessage.WriteString("errors encountered while running " + string(aRun.Type) + ":\n")
			errorMessage.WriteString(aRun.Error + "\n")
		}
		logMessage.WriteString(string(aRun.Type) + " - " + run.ID.String() + ": " + aRun.BeginDate.String() + " -> " + aRun.EndDate.String() + "\n")
	}

	// replace the lock file by error file in case of errors
	if errorMessage.Len() > 0 {
		hlog.Error(errorMessage.String())
		_, err := writeRunToFile(run, h.StorageFolder, ErrorFile)
		if err != nil {
			hlog.Fatal(err)
		}
		return
	}

	// no errors : simply log the results and rename lock to log
	hlog.Info(logMessage.String())
	_, err := writeRunToFile(run, h.StorageFolder, LogFile)
	if err != nil {
		hlog.Fatal(err)
	}
}
