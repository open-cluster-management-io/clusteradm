// Copyright Contributors to the Open Cluster Management project
package hub

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Options struct {
	ConfigFlags *genericclioptions.ConfigFlags
	factory     cmdutil.Factory
	values      map[string]interface{}
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
