// Copyright Contributors to the Open Cluster Management project
package util

import (
	"bufio"
	"fmt"
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
}

type clusteradm struct {
	h HandledOutput
}

func (adm *clusteradm) Version() error {
	fmt.Fprintln(os.Stdout, "clusteradm version ")
	return newClusteradmCmd(false, &adm.h, "version")
}

func (adm *clusteradm) Init(args ...string) error {
	fmt.Fprintln(os.Stdout, "clusteradm init ", args)
	return newClusteradmCmd(true, &adm.h, "init", args...)
}

func (adm *clusteradm) Join(args ...string) error {
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
	fmt.Fprintln(os.Stdout, "clusteradm upgrade", args)
	return newClusteradmCmd(false, &adm.h, "upgrade", args...)
}

func newClusteradmCmd(flag bool, handled *HandledOutput, subcommand string, args ...string) error {
	cmdargs := []string{subcommand}
	cmdargs = append(cmdargs, args...)
	c := exec.Command("clusteradm", cmdargs...)

	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if flag {
		out, err := c.StdoutPipe()
		if err != nil {
			return err
		}
		_ = c.Start()

		var h HandledOutput
		go func(t *HandledOutput) {
			// scan the output and find the clusteradm join command.
			scanner := bufio.NewScanner(out)
			var line string
			for scanner.Scan() {
				line = strings.TrimSpace(scanner.Text())
				fmt.Fprintln(os.Stdout, line)
				if strings.HasPrefix(line, "clusteradm") {
					*t = *handleOutput(line)
				}
			}
		}(&h)

		_ = c.Wait()
		*handled = h
		return nil
	}

	c.Stdout = os.Stdout
	return c.Run()
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
