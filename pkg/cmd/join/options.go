// Copyright Contributors to the Open Cluster Management project
package join

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

//Options: The structure holding all the command-line options
type Options struct {
	//ConfigFlags: The generic options from the kubernetes cli-runtime.
	ConfigFlags *genericclioptions.ConfigFlags
	//The token generated on the hub to access it from the cluster
	token string
	//The external hub apiserver url (https://<host>:<port>)
	hubAPIServer string
	//the name under the cluster must be imported
	clusterName string

	factory cmdutil.Factory
	values  Values
	//if set the resources will be sent to stdout instead of being applied
	dryRun bool
	//The file to output the resources will be sent to the file.
	outputFile string
}

//Values: The values used in the template
type Values struct {
	//ClusterName: the name of the joined cluster on the hub
	ClusterName string
	//Hub: Hub information
	Hub Hub
}

//Hub: The hub values for the template

type Hub struct {
	//APIServer: The API Server external URL
	APIServer string
	//KubeConfig: The kubeconfig of the boostrap secret to connect to the hub
	KubeConfig string
}

func newOptions(f cmdutil.Factory, streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),
		factory:     f,
	}
}
