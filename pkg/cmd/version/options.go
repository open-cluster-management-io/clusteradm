// Copyright Contributors to the Open Cluster Management project
package version

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Options struct {
	ConfigFlags *genericclioptions.ConfigFlags
}

func newOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
	}
}
