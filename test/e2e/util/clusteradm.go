// Copyright Contributors to the Open Cluster Management project
package util

import (
	"fmt"
	"os"
)

type clusteradmInterface interface {
	Version() outputInterface
	Init(args ...string) outputInterface
	Join(args ...string) outputInterface
	Accept(args ...string) outputInterface
	Get(args ...string) outputInterface
	Delete(args ...string) outputInterface
	Addon(args ...string) outputInterface
	Clean(args ...string) outputInterface
	Install(args ...string) outputInterface
	Proxy(args ...string) outputInterface
	Unjoin(args ...string) outputInterface
}

type clusteradm struct {
}

func (adm *clusteradm) Version() outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm version ")
	return newClusteradm("version")
}

func (adm *clusteradm) Init(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm init ", args)
	return newClusteradm("init", args...)
}

func (adm *clusteradm) Join(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm join ", args)
	return newClusteradm("join", args...)
}

func (adm *clusteradm) Accept(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm accept ", args)
	return newClusteradm("accept", args...)
}

func (adm *clusteradm) Get(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm get ", args)
	return newClusteradm("get", args...)
}

func (adm *clusteradm) Delete(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm delete ", args)
	return newClusteradm("delete", args...)
}

func (adm *clusteradm) Addon(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm addon ", args)
	return newClusteradm("addon", args...)
}

func (adm *clusteradm) Clean(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm clean ", args)
	return newClusteradm("clean", args...)
}

func (adm *clusteradm) Install(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm install ", args)
	return newClusteradm("install", args...)
}

func (adm *clusteradm) Proxy(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm proxy ", args)
	return newClusteradm("proxy", args...)
}

func (adm *clusteradm) Unjoin(args ...string) outputInterface {
	fmt.Fprintln(os.Stdout, "clusteradm unjoin ", args)
	return newClusteradm("unjoin", args...)
}
