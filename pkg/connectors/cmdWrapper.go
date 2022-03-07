package connectors

import (
	"bytes"
	"os/exec"
)

type CmdWrapper interface {
	Prepare(command string, params []string)
	CmdExec() (string, error)
}

type Connector struct {
	cmd *exec.Cmd
}

func (c *Connector) Prepare(command string, params []string) {
	c.cmd = exec.Command(command, params...)
}

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
