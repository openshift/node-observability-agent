package handlers

import (
	"os"

	"github.com/sirupsen/logrus"
)

var hlog = logrus.WithField("module", "handler")

type Handlers struct {
	Token          string
	NodeIP         string
	StorageFolder  string
	CrioUnixSocket string
}

func NewHandlers(token string, storageFolder string, crioUnixSocket string) *Handlers {
	//Get env var NODE_IP
	node := os.Getenv("NODE_IP")
	if node == "" {
		panic("Did not find environment variable $NODE_IP")
	}
	return &Handlers{
		Token:          token,
		NodeIP:         node,
		StorageFolder:  storageFolder,
		CrioUnixSocket: crioUnixSocket,
	}
}
