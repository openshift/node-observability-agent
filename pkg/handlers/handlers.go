package handlers

import (
	"bytes"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

var hlog = logrus.WithField("module", "handler")

type Handlers struct {
	Token          string
	NodeIP         string
	StorageFolder  string
	CrioUnixSocket string
}

type Run struct {
	Type      string
	Sucessful bool
	BeginDate time.Time
	EndDate   time.Time
	Error     error
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

func (h *Handlers) HandleProfiling(w http.ResponseWriter, r *http.Request) {

	runResultsChan := make(chan Run)
	go func() {

		runResultsChan <- h.ProfileKubelet()
	}()

	go func() {
		runResultsChan <- h.ProfileCrio()
	}()

	runResults := []Run{<-runResultsChan, <-runResultsChan}

	var errorMessage bytes.Buffer
	var perfMessage bytes.Buffer
	for _, aRun := range runResults {
		if aRun.Error != nil {
			errorMessage.WriteString("errors encountered while running " + aRun.Type + ":\n")
			errorMessage.WriteString(aRun.Error.Error() + "\n")
		}
		perfMessage.WriteString(aRun.Type + ": " + aRun.BeginDate.String() + " -> " + aRun.EndDate.String() + "\n")
	}
	if errorMessage.Len() > 0 {
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr := w.Write(errorMessage.Bytes())
		if writeErr != nil {
			hlog.Errorf("could not write response to requester: %v", writeErr)
		}
		hlog.Error(errorMessage.String())
		return
	}
	perfMessage.WriteString("Successfully saved results to " + h.StorageFolder)
	_, err := w.Write(perfMessage.Bytes())
	if err != nil {
		hlog.Errorf("could not write response: %v", err)
	}
}
