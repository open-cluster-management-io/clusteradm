// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	"github.com/spf13/pflag"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type ClusteradmFlagsGetter interface {
	GetClusteradmFlag() *ClusteradmFlags
}

type ClusteradmFlags struct {
	KubectlFactory cmdutil.Factory
	//if set the resources will be sent to stdout instead of being applied
	DryRun bool
}

// NewClusteradmFlags returns ClusteradmFlags with default values set
func NewClusteradmFlags(f cmdutil.Factory) *ClusteradmFlags {
	return &ClusteradmFlags{
		KubectlFactory: f,
	}
}

func (f *ClusteradmFlags) AddFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&f.DryRun, "dry-run", false, "If set the generated resources will be displayed but not applied")
}
