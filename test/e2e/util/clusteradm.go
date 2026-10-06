// Copyright Contributors to the Open Cluster Management project
package util

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type clusteradmInterface interface {
	Version() error
	Init(args ...string) error
	Join(args ...string) error
	Accept(args ...string) error
	Get(args ...string) error
	Delete(args ...string) error
	Addon(args ...string) error
	Clean(args ...string) error
	Install(args ...string) error
	Proxy(args ...string) error
	Unjoin(args ...string) error
	Upgrade(args ...string) error

	WithVersion(string) clusteradmInterface
	Result() *HandledOutput
}

type clusteradm struct {
	h       HandledOutput
	version string
}

func (adm *clusteradm) WithVersion(version string) clusteradmInterface {
	adm.version = version
	return adm
}

func (adm *clusteradm) Version() error {
	fmt.Fprintln(os.Stdout, "clusteradm version ")
	return newClusteradmCmd(false, &adm.h, "version")
}

func (adm *clusteradm) Init(args ...string) error {
	if adm.version != "" {
		args = append(args, "--bundle-version="+adm.version)
	}
	fmt.Fprintln(os.Stdout, "clusteradm init ", args)
	return newClusteradmCmd(true, &adm.h, "init", args...)
}

func (adm *clusteradm) Join(args ...string) error {
	if adm.version != "" {
		args = append(args, "--bundle-version="+adm.version)
	}
	fmt.Fprintln(os.Stdout, "clusteradm join ", args)
	return newClusteradmCmd(false, &adm.h, "join", args...)
}

func (adm *clusteradm) Accept(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm accept ", args)
	return newClusteradmCmd(false, &adm.h, "accept", args...)
}

func (adm *clusteradm) Get(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm get ", args)
	return newClusteradmCmd(true, &adm.h, "get", args...)
}

func (adm *clusteradm) Delete(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm delete ", args)
	return newClusteradmCmd(false, &adm.h, "delete", args...)
}

func (adm *clusteradm) Addon(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm addon ", args)
	return newClusteradmCmd(false, &adm.h, "addon", args...)
}

func (adm *clusteradm) Clean(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm clean ", args)
	return newClusteradmCmd(false, &adm.h, "clean", args...)
}

func (adm *clusteradm) Install(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm install ", args)
	return newClusteradmCmd(false, &adm.h, "install", args...)
}

func (adm *clusteradm) Proxy(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm proxy ", args)
	return newClusteradmCmd(false, &adm.h, "proxy", args...)
}

func (adm *clusteradm) Unjoin(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm unjoin ", args)
	return newClusteradmCmd(false, &adm.h, "unjoin", args...)
}

func (adm *clusteradm) Upgrade(args ...string) error {
	if adm.version != "" {
		args = append(args, "--bundle-version="+adm.version)
	}
	fmt.Fprintln(os.Stdout, "clusteradm upgrade", args)
	return newClusteradmCmd(false, &adm.h, "upgrade", args...)
}

func (adm *clusteradm) Result() *HandledOutput {
	return &adm.h
}

func newClusteradmCmd(captureStdout bool, handled *HandledOutput, subcommand string, args ...string) error {
	cmdargs := make([]string, 0, 1+len(args))
	cmdargs = append(cmdargs, subcommand)
	cmdargs = append(cmdargs, args...)
	c := exec.Command("clusteradm", cmdargs...)

	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if !captureStdout {
		c.Stdout = os.Stdout
		return c.Run()
	}

	var stderr bytes.Buffer
	c.Stderr = io.MultiWriter(os.Stderr, &stderr)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	if err := c.Start(); err != nil {
		return err
	}

	// StdoutPipe must be fully read before Wait.
	// Wait does not synchronize with readers.
	output, scanErr := scanHandledOutput(stdout)
	if scanErr != nil {
		_ = stdout.Close()
		_ = c.Process.Kill()
	}
	waitErr := c.Wait()
	*handled = output
	if scanErr != nil {
		return scanErr
	}
	return commandError(waitErr, stderr.String())
}

func commandError(err error, stderr string) error {
	if err == nil {
		return nil
	}
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, stderr)
}

func scanHandledOutput(r io.Reader) (HandledOutput, error) {
	var handled HandledOutput
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fmt.Fprintln(os.Stdout, line)
		if strings.HasPrefix(line, "clusteradm") {
			handled = *handleOutput(line)
		}
	}
	return handled, scanner.Err()
}

func handleOutput(content string) *HandledOutput {
	o := strings.Split(content, " ")
	return &HandledOutput{
		raw:   content,
		token: o[3],
		host:  o[5],
	}
}

type HandledOutput struct {
	// raw stores the raw command
	raw   string
	host  string
	token string
}

// RawCommand return the clusteradm join command
func (h *HandledOutput) RawCommand() string {
	return h.raw
}

// Host return the hub-apiserver
func (h *HandledOutput) Host() string {
	return h.host
}

// Token return the hub-token
func (h *HandledOutput) Token() string {
	return h.token
}
