//build: +fake

package connectors

import "os/exec"

// FakeConnector is a structure that holds the shell command to fake-run, its parameters
// as well as a Flag, that helps orient the behavior of the mock
type FakeConnector struct {
	command string
	params  []string
	Flag    ErrorFlag
}

// ErrorFlag values can be NoError, SocketErr or WriteErr
type ErrorFlag string

const (
	// NoError instructs the FakeConnector to return without errors
	NoError ErrorFlag = ""
	// SocketErr instructs the FakeConnector to return an error related to the curl command execution
	SocketErr ErrorFlag = "socket-error"
	// WriteErr instructs the FakeConnector to return an error related to the storage of curl output to file
	WriteErr ErrorFlag = "write-error"
)

// Prepare stores the command and parameters to be used by FakeConnector, preparing the call to CmdExec
// Implementation of cmdWrapper.Prepare
func (c *FakeConnector) Prepare(command string, params []string) {
	c.command = command
	c.params = params
}

// CmdExec implements the cmdWrapper.CmdExec and returns fake responses based on FakeConnector.Flag
func (c *FakeConnector) CmdExec() (string, error) {
	if c.Flag == SocketErr {
		return "curl: (7) Couldn't connect to server", &exec.ExitError{}
	}

	if c.Flag == WriteErr {
		return "curl: (23) Failure writing output to destination", &exec.ExitError{}
	}
	return "", nil
}
