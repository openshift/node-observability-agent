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
	"github.com/openshift/node-observability-agent/pkg/turntaker"
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
	turnTaker turntaker.SingleTaker
}

type fileType string

// NewHandlers creates a new instance of Handlers from the given parameters
func NewHandlers(token string, storageFolder string, crioUnixSocket string, nodeIP string) *Handlers {
	aTurnTaker := turntaker.NewTurnTaker(filepath.Join(storageFolder, "agent."+string(errorFile)))
	return &Handlers{
		Token:          token,
		NodeIP:         nodeIP,
		StorageFolder:  storageFolder,
		CrioUnixSocket: crioUnixSocket,
		turnTaker:      aTurnTaker,
	}
}

const (
	ready              = "Service is ready"
	timeout   int      = 35
	logFile   fileType = "log"
	errorFile fileType = "err"
)

// Status is called when the agent receives an HTTP request on endpoint /status.
// It returns:
// * HTTP 500 if the agent is in error,
// * HTTP 409 if a previous profiling is still ongoing,
// * HTTP 200 if the agent is ready
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	id, state, err := h.turnTaker.WhoseTurn()
	if err != nil {
		http.Error(w, "error retrieving service status",
			http.StatusInternalServerError)
		hlog.Errorf("Error retrieving service status : %v", err)
		return
	}
	switch state {
	case turntaker.InError:
		err = respondBusyOrError(id.String(), w, r, true)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
		return
	case turntaker.Taken:
		err := respondBusyOrError(id.String(), w, r, false)
		if err != nil {
			http.Error(w, "unable to send response",
				http.StatusInternalServerError)
			hlog.Error("unable to send response")
			return
		}
		return
	case turntaker.Free:
		_, err := w.Write([]byte(ready))
		if err != nil {
			hlog.Errorf("could not send response busy : %v", err)
		}
	}
}

func sendUID(w http.ResponseWriter, r *http.Request, runID uuid.UUID) (runs.Run, error) {
	response := runs.Run{
		ID: runID,
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
	return response, nil
}

func respondBusyOrError(uid string, w http.ResponseWriter, r *http.Request, isError bool) error {

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
	uidWon, currentUID, state, err := h.turnTaker.TakeTurn()
	if err != nil {
		http.Error(w, "service is either busy or in error, try again",
			http.StatusInternalServerError)
		hlog.Error(err)
		return
	}

	switch state {
	case turntaker.InError:
		{
			err := respondBusyOrError(currentUID.String(), w, r, true)
			if err != nil {
				http.Error(w, "unable to send response",
					http.StatusInternalServerError)
				hlog.Error("unable to send response")
				return
			}
			return
		}
	case turntaker.Taken:
		{
			err := respondBusyOrError(currentUID.String(), w, r, false)
			if err != nil {
				http.Error(w, "unable to send response",
					http.StatusInternalServerError)
				hlog.Error("unable to send response")
				return
			}
			return
		}
	case turntaker.Free:
		{
			// Send a HTTP 200 straight away
			run, err := sendUID(w, r, uidWon)
			if err != nil {
				hlog.Error(err)
				return
			}
			// Channel for collecting results of profiling
			runResultsChan := make(chan runs.ProfilingRun)

			// Launch both profilings in parallel as well as the routine to wait for results
			go func() {
				//TODO Go back to securely making this request
				//Prepare http client that ignores tls check
				transCfg := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: transCfg}
				runResultsChan <- h.profileKubelet(run.ID.String(), client)
			}()

			go func() {
				connector := connectors.Connector{}
				connector.Prepare("curl", []string{"--unix-socket", h.CrioUnixSocket, "http://localhost/debug/pprof/profile", "--output", filepath.Join(h.StorageFolder, "crio-"+run.ID.String()+".pprof")})
				runResultsChan <- h.profileCrio(run.ID.String(), &connector)
			}()

			go h.processResults(run, runResultsChan)
		}
	}

}
func (h *Handlers) processResults(arun runs.Run, runResultsChan chan runs.ProfilingRun) {
	// unlock as soon as finished processing
	defer func() {
		err := h.turnTaker.ReleaseTurn(arun)
		if err != nil {
			hlog.Fatal(err)
		}
		close(runResultsChan)
	}()
	// wait for the results
	arun.ProfilingRuns = []runs.ProfilingRun{}
	isTimeout := false
	for nb := 0; nb < 2 && !isTimeout; {
		select {
		case pr := <-runResultsChan:
			nb++
			arun.ProfilingRuns = append(arun.ProfilingRuns, pr)
		case <-time.After(time.Second * time.Duration(timeout)):
			//timeout! dont wait anymore
			prInTimeout := runs.ProfilingRun{
				Type:       runs.UnknownRun,
				Successful: false,
				BeginDate:  time.Now(),
				EndDate:    time.Now(),
				Error:      fmt.Sprintf("timeout after waiting %ds", timeout),
			}
			prInTimeout.Error = fmt.Sprintf("timeout after waiting %ds", timeout)
			arun.ProfilingRuns = append(arun.ProfilingRuns, prInTimeout)
			isTimeout = true
		}
	}

	// Process the results
	var errorMessage bytes.Buffer
	var logMessage bytes.Buffer
	for _, aProfRun := range arun.ProfilingRuns {
		if aProfRun.Error != "" {
			errorMessage.WriteString("errors encountered while running " + string(aProfRun.Type) + ":\n")
			errorMessage.WriteString(aProfRun.Error + "\n")
		}
		logMessage.WriteString(string(aProfRun.Type) + " - " + arun.ID.String() + ": " + aProfRun.BeginDate.String() + " -> " + aProfRun.EndDate.String() + "\n")
	}

	if errorMessage.Len() > 0 {
		hlog.Error(errorMessage.String())
		return
	} else {
		// no errors : simply log the results and rename lock to log
		hlog.Info(logMessage.String())
		_, err := writeRunToLogFile(arun, h.StorageFolder)
		//important to clear the run so that ReleaseTurn doesnt generate an error file
		arun = runs.Run{}
		if err != nil {
			hlog.Fatal(err)
		}
	}

}

func writeRunToLogFile(arun runs.Run, storageFolder string) (string, error) {

	fileName := filepath.Join(storageFolder, arun.ID.String()+"."+string(logFile))

	bytes, err := json.Marshal(arun)
	if err != nil {
		return "", fmt.Errorf("error while creating %s file : unable to marshal run of ID %s\n%v", string(logFile), arun.ID.String(), err)
	}
	err = os.WriteFile(fileName, bytes, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing  %s file: %v", fileName, err)
	}
	return fileName, nil
}
