// Copyright Contributors to the Open Cluster Management project
package init

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Options struct {
	ConfigFlags *genericclioptions.ConfigFlags
	factory     cmdutil.Factory
	values      Values
}

type Values struct {
	Hub Hub `json:"hub"`
}

type Hub struct {
	TokenID     string `json:"tokenID"`
	TokenSecret string `json:"tokenSecret"`
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
