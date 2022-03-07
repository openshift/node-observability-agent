//build: +fake

package connectors

import "os/exec"

type MockConnector struct {
	command string
	params  []string
	Flag    ErrorFlag
}

type ErrorFlag string

const (
	NO_ERROR   ErrorFlag = ""
	SOCKET_ERR ErrorFlag = "socket-error"
	WRITE_ERR  ErrorFlag = "write-error"
)

func (c *MockConnector) Prepare(command string, params []string) {
	c.command = command
	c.params = params
}

func (c *MockConnector) CmdExec() (string, error) {
	if c.Flag == SOCKET_ERR {
		return "curl: (7) Couldn't connect to server", &exec.ExitError{}
	}

	if c.Flag == WRITE_ERR {
		return "curl: (23) Failure writing output to destination", &exec.ExitError{}
	}
	return "", nil
}
