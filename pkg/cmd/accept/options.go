// Copyright Contributors to the Open Cluster Management project
package accept

import (
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Options struct {
	//ConfigFlags: The generic options from the kubernetes cli-runtime.
	ConfigFlags *genericclioptions.ConfigFlags
	factory     cmdutil.Factory
	//A list of comma separated cluster names
	clusters string
	values   Values
}

//Values: The values used in the template
type Values struct {
	clusters []string
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
