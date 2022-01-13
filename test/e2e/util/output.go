// Copyright Contributors to the Open Cluster Management project
package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type outputInterface interface {
	Run() error
	// if the command output is need to be reuse, for example, init's which includes token and hub-apiserver,
	// just call Output() rather than Run(), and it will return a handledOutput.
	Output() (*handledOutput, error)
}

type command struct {
	cmd *exec.Cmd
}

func newClusteradm(subcommand string, args ...string) *command {
	cmdargs := []string{subcommand}
	cmdargs = append(cmdargs, args...)

	return &command{
		cmd: exec.Command("clusteradm", cmdargs...),
	}
}

// Run execute the command and print the result to stdout.
func (c *command) Run() error {
	c.cmd.Stdin = os.Stdin
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	return c.cmd.Run()
}

func (c *command) Output() (handled *handledOutput, err error) {
	c.cmd.Stdin = os.Stdin
	c.cmd.Stderr = os.Stderr
	out, err := c.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	c.cmd.Start()

	go func() {
		// scan the output and find the clusteradm join command.
		scanner := bufio.NewScanner(out)
		var line string
		for scanner.Scan() {
			line = strings.TrimSpace(scanner.Text())
			fmt.Fprintln(os.Stdout, line)
			if strings.HasPrefix(line, "clusteradm") {
				handled = handleOutput(line)
			}
		}
	}()

	c.cmd.Wait()
	return handled, nil
}

type handledOutput struct {
	// raw stores the raw command
	raw   string
	host  string
	token string
}

// RawCommand return the clusteradm join command
func (h *handledOutput) RawCommand() string {
	if h == nil {
		return ""
	}
	return h.raw
}

// Host return the hub-apiserver
func (h *handledOutput) Host() string {
	if h == nil {
		return ""
	}
	return h.host
}

// Token return the hub-token
func (h *handledOutput) Token() string {
	if h == nil {
		return ""
	}
	return h.token
}

func handleOutput(command string) *handledOutput {
	if len(command) == 0 {
		return &handledOutput{}
	}
	o := strings.Split(command, " ")
	return &handledOutput{
		raw:   command,
		token: o[3],
		host:  o[5],
	}
}
