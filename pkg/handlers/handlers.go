package handlers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/openshift/node-observability-agent/pkg/connectors"
	"github.com/openshift/node-observability-agent/pkg/runs"
	"github.com/openshift/node-observability-agent/pkg/statelocker"
)

var hlog = logrus.WithField("module", "handler")

// Handlers holds the parameters necessary for running the CRIO and Kubelet profiling
type Handlers struct {
	Token          string
	NodeIP         string
	StorageFolder  string
	CrioUnixSocket string
	// mux            *sync.Mutex
	// onGoingRunID   string
	stateLocker statelocker.StateLocker
}

type fileType string

// NewHandlers creates a new instance of Handlers from the given parameters
func NewHandlers(token string, storageFolder string, crioUnixSocket string, nodeIP string) *Handlers {
	aStateLocker := statelocker.NewStateLock(filepath.Join(storageFolder, "agent."+string(errorFile)))
	return &Handlers{
		Token:          token,
		NodeIP:         nodeIP,
		StorageFolder:  storageFolder,
		CrioUnixSocket: crioUnixSocket,
		stateLocker:    aStateLocker,
	}
}

// NewMetricsHandlers creates a new instance of Handlers from the given parameters
func NewMetricsHandlers(storageFolder string, nodeIP string) *Handlers {
	aStateLocker := statelocker.NewStateLock(filepath.Join(storageFolder, "agent."+string(errorFile)))
	return &Handlers{
		NodeIP:        nodeIP,
		StorageFolder: storageFolder,
		stateLocker:   aStateLocker,
	}
}

const (
	ready                   = "Service is ready"
	httpRespErrMsg          = "unable to send response"
	baseTimeout    int      = 35
	logFile        fileType = "log"
	errorFile      fileType = "err"
)

// Status is called when the agent receives an HTTP request on endpoint /status.
// It returns:
// * HTTP 500 if the agent is in error,
// * HTTP 409 if a previous profiling is still ongoing,
// * HTTP 200 if the agent is ready
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	id, state, err := h.stateLocker.LockInfo()
	if err != nil {
		http.Error(w, "error retrieving service status",
			http.StatusInternalServerError)
		hlog.Errorf("Error retrieving service status : %v", err)
		return
	}
	switch state {
	case statelocker.InError:
		err = respondBusyOrError(id.String(), w, true)
		if err != nil {
			http.Error(w, httpRespErrMsg,
				http.StatusInternalServerError)
			hlog.Error(httpRespErrMsg)
			return
		}
	case statelocker.Taken:
		err := respondBusyOrError(id.String(), w, false)
		if err != nil {
			http.Error(w, httpRespErrMsg,
				http.StatusInternalServerError)
			hlog.Error(httpRespErrMsg)
			return
		}
	case statelocker.Free:
		_, err := w.Write([]byte(ready))
		if err != nil {
			hlog.Errorf("could not send response busy : %v", err)
		}
	}
}

func sendUID(w http.ResponseWriter, runID uuid.UUID) error {
	response := runs.Run{
		ID: runID,
	}

	jsResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsResponse)
	if err != nil {
		hlog.Errorf("Unable to send HTTP response : %v", err)
	}
	return nil
}

func respondBusyOrError(uid string, w http.ResponseWriter, isError bool) error {

	message := ""

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
	return nil
}

// HandleProfiling is called when the agent receives an HTTP request on endpoint /pprof
// After checking the agent is not in error, and that no previous profiling is still ongoing,
// it triggers the kubelet and CRIO profiling in separate goroutines, and launches a separate
// function to process the results in a goroutine as well
func (h *Handlers) HandleProfiling(w http.ResponseWriter, r *http.Request) {
	uid, state, err := h.stateLocker.Lock()
	if err != nil {
		http.Error(w, "service is either busy or in error, try again",
			http.StatusInternalServerError)
		hlog.Error(err)
		return
	}

	switch state {
	case statelocker.InError:
		{
			err := respondBusyOrError(uid.String(), w, true)
			if err != nil {
				http.Error(w, httpRespErrMsg,
					http.StatusInternalServerError)
				hlog.Error(httpRespErrMsg)
				return
			}
			return
		}
	case statelocker.Taken:
		{
			err := respondBusyOrError(uid.String(), w, false)
			if err != nil {
				http.Error(w, httpRespErrMsg,
					http.StatusInternalServerError)
				hlog.Error(httpRespErrMsg)
				return
			}
			return
		}
	case statelocker.Free:
		{

			// Channel for collecting results of profiling
			runResultsChan := make(chan runs.ExecutionRun)

			// Launch both profilings in parallel as well as the routine to wait for results
			go func() {
				//TODO Go back to securely making this request
				//Prepare http client that ignores tls check
				transCfg := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: transCfg}
				runResultsChan <- h.profileKubelet(uid.String(), client)
			}()

			go func() {
				connector := connectors.Connector{}
				connector.Prepare("curl", []string{"--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", filepath.Join(h.StorageFolder, "crio-"+uid.String()+".pprof")})
				runResultsChan <- h.profileCrio(uid.String(), &connector)
			}()

			go h.processResults(uid, runResultsChan, baseTimeout)
			// Send a HTTP 200 straight away
			err := sendUID(w, uid)
			if err != nil {
				hlog.Error(err)
				return
			}
		}
	}
}

// HandleMetrics is called when the agent receives an HTTP request on endpoint /metrics
// After checking the agent is not in error, and that no previous profiling is still ongoing,
// it triggers the embedded script in a separate goroutines, and launches a separate
// function to process the results in a goroutine as well
func (h *Handlers) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	uid, state, err := h.stateLocker.Lock()
	if err != nil {
		http.Error(w, "service is either busy or in error, try again",
			http.StatusInternalServerError)
		hlog.Error(err)
		return
	}

	switch state {
	case statelocker.InError:
		{
			err := respondBusyOrError(uid.String(), w, true)
			if err != nil {
				http.Error(w, httpRespErrMsg,
					http.StatusInternalServerError)
				hlog.Error(httpRespErrMsg)
				return
			}
			return
		}
	case statelocker.Taken:
		{
			err := respondBusyOrError(uid.String(), w, false)
			if err != nil {
				http.Error(w, httpRespErrMsg,
					http.StatusInternalServerError)
				hlog.Error(httpRespErrMsg)
				return
			}
			return
		}
	case statelocker.Free:
		{

			// Channel for collecting results of metrics
			runResultsChan := make(chan runs.ExecutionRun)

			// Launch metrics script as the routine to wait for results
			go func() {
				connector := connectors.Connector{}
				connector.Prepare("sh", []string{"-c", "scripts/all_metrics.sh"})
				runResultsChan <- h.metrics(uid.String(), &connector)
			}()

			go h.processResults(uid, runResultsChan, 7200)
			// Send a HTTP 200 straight away
			err := sendUID(w, uid)
			if err != nil {
				hlog.Error(err)
				return
			}
		}
	}
}

func (h *Handlers) processResults(uid uuid.UUID, runResultsChan chan runs.ExecutionRun, timeout int) {
	arun := runs.Run{
		ID:            uid,
		ProfilingRuns: []runs.ExecutionRun{},
	}
	// unlock as soon as finished processing
	defer func() {
		err := h.stateLocker.Unlock()
		if err != nil {
			hlog.Fatal(err)
		}
		close(runResultsChan)
	}()
	// wait for the results
	arun.ProfilingRuns = []runs.ExecutionRun{}
	//isTimeout := false
	//for nb := 0; nb < 2 && !isTimeout; {
	select {
	case pr := <-runResultsChan:
		//nb++
		arun.ProfilingRuns = append(arun.ProfilingRuns, pr)
	case <-time.After(time.Second * time.Duration(timeout)):
		//timeout! dont wait anymore
		prInTimeout := runs.ExecutionRun{
			Type:       runs.UnknownRun,
			Successful: false,
			BeginTime:  time.Now(),
			EndTime:    time.Now(),
			Error:      fmt.Sprintf("timeout after waiting %ds", timeout),
		}
		prInTimeout.Error = fmt.Sprintf("timeout after waiting %ds", timeout)
		arun.ProfilingRuns = append(arun.ProfilingRuns, prInTimeout)
		//isTimeout = true
	}
	//}

	// Process the results
	var errorMessage bytes.Buffer
	var logMessage bytes.Buffer
	for _, profRun := range arun.ProfilingRuns {
		if profRun.Error != "" {
			errorMessage.WriteString("errors encountered while running " + string(profRun.Type) + ":\n")
			errorMessage.WriteString(profRun.Error + "\n")
		}
		logMessage.WriteString(string(profRun.Type) + " - " + arun.ID.String() + ": " + profRun.BeginTime.String() + " -> " + profRun.EndTime.String() + "\n")
	}

	if errorMessage.Len() > 0 {
		hlog.Error(errorMessage.String())
		err := h.stateLocker.SetError(arun)
		if err != nil {
			hlog.Fatal(err)
		}
	} else {
		// no errors : simply log the results
		hlog.Info(logMessage.String())
		_, err := writeRunToLogFile(arun, h.StorageFolder)
		if err != nil {
			hlog.Fatal(err)
		}
	}
}

func writeRunToLogFile(arun runs.Run, storageFolder string) (string, error) {

	fileName := filepath.Join(storageFolder, arun.ID.String()+"."+string(logFile))

	bytes, err := json.Marshal(arun)
	if err != nil {
		return "", fmt.Errorf("error while creating %s file : unable to marshal run of ID %s\n%w", string(logFile), arun.ID.String(), err)
	}
	err = os.WriteFile(fileName, bytes, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing  %s file: %w", fileName, err)
	}
	return fileName, nil
}
