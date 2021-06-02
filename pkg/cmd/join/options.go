// Copyright Contributors to the Open Cluster Management project
package join

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Options struct {
	ConfigFlags       *genericclioptions.ConfigFlags
	token             string
	hubServerExternal string
	hubServerInternal string
	clusterName       string
	factory           cmdutil.Factory
	values            Values
}

type Values struct {
	ClusterName string
	Hub         Hub
}

type Hub struct {
	ExternalServerURL string
	InternalServerURL string
	KubeConfig        string
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
