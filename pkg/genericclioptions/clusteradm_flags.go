// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	"fmt"
	"strconv"

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

const minTimeout = 30

type minIntValue struct {
	p   *int
	min int
}

func (v *minIntValue) String() string {
	return strconv.Itoa(*v.p)
}

func (v *minIntValue) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if n < v.min {
		return fmt.Errorf("must be at least %d", v.min)
	}
	*v.p = n
	return nil
}

func (v *minIntValue) Type() string { return "int" }

// NewClusteradmFlags returns ClusteradmFlags with default values set
func NewClusteradmFlags(f cmdutil.Factory) *ClusteradmFlags {
	return &ClusteradmFlags{
		KubectlFactory: f,
	}
}

func (f *ClusteradmFlags) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&f.DryRun, "dry-run", false, "If set the generated resources will be displayed but not applied")
	f.Timeout = 300
	flags.Var(&minIntValue{&f.Timeout, minTimeout}, "timeout", "set the timeout deadline for the command in seconds")
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
	return check.CheckForHub(client.Discovery())
}
func (f *ClusteradmFlags) ValidateManagedCluster() error {
	client, err := f.buildClusterClientset()
	if err != nil {
		return err
	}
	return check.CheckForManagedCluster(client.Discovery())
}

func (f *ClusteradmFlags) buildClusterClientset() (*clusterclientset.Clientset, error) {
	config, err := f.KubectlFactory.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("build ClusteradmFlags failed: %v", err)
	}
	client, err := clusterclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("build ClusteradmFlags failed: %v", err)
	}
	return client, nil
}
