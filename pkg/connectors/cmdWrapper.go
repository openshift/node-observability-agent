package connectors

import (
	"bytes"
	"os/exec"
)

// CmdWrapper wrap a exec.Cmd, or its fake
type CmdWrapper interface {
	// Prepare sets the command and its parameters that are wrapped by CmdWrapper
	Prepare(command string, params []string)
	// CmdExec executes the command wrapped by CmdWrapper
	CmdExec() (string, error)
}

// Connector represents a command being prepared to be run
type Connector struct {
	cmd *exec.Cmd
}

// Prepare sets the command and parameters to be called
func (c *Connector) Prepare(command string, params []string) {
	c.cmd = exec.Command(command, params...)
}

// CmdExec runs the command on the underlying system and returns the stdout or stderr as a string
func (c *Connector) CmdExec() (string, error) {
	var stdout, stderr bytes.Buffer
	c.cmd.Stdout = &stdout
	c.cmd.Stderr = &stderr
	err := c.cmd.Run()
	outStr, errStr := stdout.String(), stderr.String()
	if err != nil {
		return errStr, err
	}
	return outStr, nil
}
