// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Init the hub
%[1]s init
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a Kubernetes cluster into an OCM hub cluster.",
		Long: "Initialize the Kubernetes cluster in the context into an OCM hub cluster by applying a few" +
			" fundamental resources including registration-operator, etc.",
		Example:      fmt.Sprintf(example, helpers.GetExampleHeader()),
		SilenceUsage: true,
		PreRun: func(c *cobra.Command, args []string) {
			helpers.DryRunMessage(o.ClusteradmFlags.DryRun)
		},
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	genericclioptionsclusteradm.HubMutableFeatureGate.AddFlag(cmd.Flags())
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().BoolVar(&o.useBootstrapToken, "use-bootstrap-token", false, "If set then the bootstrap token will used instead of a service account token")
	cmd.Flags().BoolVar(&o.force, "force", false, "If set then the hub will be reinitialized")
	cmd.Flags().StringVar(&o.registry, "image-registry", "quay.io/open-cluster-management",
		"The name of the image registry serving OCM images, which will be applied to all the deploying OCM components.")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")
	cmd.Flags().StringVar(&o.outputJoinCommandFile, "output-join-command-file", "",
		"If set, the generated join command be saved to the prescribed file.")
	cmd.Flags().BoolVar(&o.wait, "wait", false,
		"If set, the command will initialize the OCM control plan in foreground.")
	cmd.Flags().StringVarP(&o.output, "output", "o", "text", "output foramt, should be json or text")
	return cmd
}
