// Copyright Red Hat
package genericclioptions

import (
	"github.com/spf13/pflag"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type ApplierFlags struct {
	KubectlFactory cmdutil.Factory
	//if set the resources will be sent to stdout instead of being applied
	DryRun  bool
	Timeout int
}

// NewApplierFlags returns ApplierFlags with default values set
func NewApplierFlags(f cmdutil.Factory) *ApplierFlags {
	return &ApplierFlags{
		KubectlFactory: f,
	}
}

// placeHolder to add generic flags for the applier
// Dryrun and Timeout options are not used in all commands (ie:render)
func (f *ApplierFlags) AddFlags(flags *pflag.FlagSet) {
}
