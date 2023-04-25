// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	cmd.Flags().BoolVar(&o.singleton, "singleton", false, "If true, deploy singleton controlplane instead of cluster-manager. This is an alpha stage flag.")

	//clusterManagetSet contains the flags for deploy cluster-manager
	clusterManagerSet := pflag.NewFlagSet("clusterManagerSet", pflag.ExitOnError)
	clusterManagerSet.BoolVar(&o.useBootstrapToken, "use-bootstrap-token", false, "If set then the bootstrap token will used instead of a service account token")
	_ = clusterManagerSet.SetAnnotation("use-bootstrap-token", "clusterManagerSet", []string{})
	cmd.Flags().AddFlagSet(clusterManagerSet)

	//singletonSet contains the flags for deploy singleton controlplane
	singletonSet := pflag.NewFlagSet("singletonSet", pflag.ExitOnError)
	singletonSet.StringVar(&o.singletonValues.controlplaneName, "name", "open-cluster-management-hub", "")

	singletonSet.StringVar(&o.singletonValues.autoApprovalBootstrapUsers, "auto-approval-bootstrap-users", "", "")
	singletonSet.BoolVar(&o.singletonValues.enableSelfManagement, "enable-self-management", false, "")
	singletonSet.BoolVar(&o.singletonValues.enableDelegatingAuthentication, "enable-delegating-authentication", false, "")
	//apiserver options
	singletonSet.StringVar(&o.singletonValues.apiserverExternalHostname, "apiserver-external-hostname", "", "")
	singletonSet.StringVar(&o.singletonValues.apiserverCA, "apiserver-ca", "", "")
	singletonSet.StringVar(&o.singletonValues.apiserverCAKey, "apiserver-ca-key", "", "")
	//etcd options
	singletonSet.StringVar(&o.singletonValues.etcdMode, "etcd-mode", "embed", "")
	singletonSet.StringSliceVar(&o.singletonValues.etcdServers, "etcd-servers", []string{"http://127.0.0.1:2379"}, "")
	singletonSet.StringVar(&o.singletonValues.etcdCA, "etcd-ca", "", "")
	singletonSet.StringVar(&o.singletonValues.etcdClientCert, "etcd-client-cert", "", "")
	singletonSet.StringVar(&o.singletonValues.etcdClientCertKey, "etcd-client-cert-key", "", "")
	//pvc options
	singletonSet.StringVar(&o.singletonValues.pvcStorageClassName, "pvc-storageclass-name", "gp2", "")
	//expose service options
	singletonSet.BoolVar(&o.singletonValues.routeEnabled, "route-enabled", true, "")
	singletonSet.BoolVar(&o.singletonValues.loadBalancerEnabled, "load-balancer-enabled", false, "")
	singletonSet.StringVar(&o.singletonValues.loadBalancerBaseDomain, "load-balancer-base-domain", "", "")
	singletonSet.BoolVar(&o.singletonValues.nodeportEnabled, "nodeport-enabled", false, "")
	singletonSet.Int16Var(&o.singletonValues.nodeportValue, "nodeport-value", 30443, "")
	//set annotions
	_ = singletonSet.SetAnnotation("name", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("auto-approval-bootstrap-users", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("enable-self-management", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("enable-delegating-authentication", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("apiserver-external-hostname", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("apiserver-ca", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("apiserver-ca-key", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("etcd-mode", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("etcd-servers", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("etcd-ca", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("etcd-client-cert", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("etcd-client-cert-key", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("pvc-storageclass-name", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("route-enabled", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("load-balancer-enabled", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("load-balancer-base-domain", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("nodeport-enabled", "singletonSet", []string{})
	_ = singletonSet.SetAnnotation("nodeport-value", "singletonSet", []string{})

	cmd.Flags().AddFlagSet(singletonSet)
	return cmd
}
