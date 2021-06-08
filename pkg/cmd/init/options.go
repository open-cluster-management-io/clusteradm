// Copyright Contributors to the Open Cluster Management project
package init

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ConfigFlags: The generic options from the kubernetes cli-runtime.
	ConfigFlags *genericclioptions.ConfigFlags
	factory     cmdutil.Factory
	values      Values
	//if set the resources will be sent to stdout instead of being applied
	dryRun bool
	//The file to output the resources will be sent to the file.
	outputFile string
}

//Valus: The values used in the template
type Values struct {
	//The values related to the hub
	Hub Hub `json:"hub"`
}

//Hub: The hub values for the template
type Hub struct {
	//TokenID: A token id allowing the cluster to connect back to the hub
	TokenID string `json:"tokenID"`
	//TokenSecret: A token secret allowing the cluster to connect back to the hub
	TokenSecret string `json:"tokenSecret"`
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
