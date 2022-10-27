// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	"fmt"

	"github.com/spf13/pflag"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers/check"
)

type ClusteradmFlags struct {
	KubectlFactory cmdutil.Factory
	//if set the resources will be sent to stdout instead of being applied
	DryRun  bool
	Timeout int
	Context string
}

// NewClusteradmFlags returns ClusteradmFlags with default values set
func NewClusteradmFlags(f cmdutil.Factory) *ClusteradmFlags {
	return &ClusteradmFlags{
		KubectlFactory: f,
	}
}

func (f *ClusteradmFlags) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&f.DryRun, "dry-run", false, "If set the generated resources will be displayed but not applied")
	flags.IntVar(&f.Timeout, "timeout", 300, "extend timeout from 300 secounds ")
}

// SetContext will set current context from command line argument --context.
func (f *ClusteradmFlags) SetContext(context *string) {
	if context != nil {
		f.Context = *context
	}
}

func (f *ClusteradmFlags) ValidateHub() error {
	client, err := f.buildClusterClientset()
	if err != nil {
		return err
	}
	return check.CheckForHub(client)
}
func (f *ClusteradmFlags) ValidateManagedCluster() error {
	client, err := f.buildClusterClientset()
	if err != nil {
		return err
	}
	return check.CheckForManagedCluster(client)
}

func (f *ClusteradmFlags) buildClusterClientset() (*clusterclientset.Clientset, error) {
	config, err := f.KubectlFactory.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("Build ClusteradmFlags failed: %v", err)
	}
	client, err := clusterclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Build ClusteradmFlags failed: %v", err)
	}
	return client, nil
}
