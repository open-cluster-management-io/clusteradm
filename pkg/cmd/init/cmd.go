// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Init the hub
%[1]s init

# Initialize the hub cluster with the type of authentication. Either or both of csr,awsirsa
%[1]s init --registration-drivers "awsirsa,csr"
    --hubClusterArn arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster1
	--aws-resource-tags product:v1:tenant:app-name=My-App,product:v1:tenant:created-by=Team-1
    --auto-approved-csr-identities="user1,user2"
	--auto-approved-arn-patterns="arn:aws:eks:us-west-2:123456789013:cluster/.*,arn:aws:eks:us-west-2:123456789012:cluster/.*"
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
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
	cmd.Flags().StringVar(&o.outputJoinCommandFile, "output-join-command-file", "",
		"If set, the generated join command be saved to the prescribed file.")
	cmd.Flags().BoolVar(&o.wait, "wait", false,
		"If set, the command will initialize the OCM control plan in foreground.")
	cmd.Flags().StringVarP(&o.output, "output", "o", "text", "output foramt, should be json or text")
	cmd.Flags().BoolVar(&o.singleton, "singleton", false, "If true, deploy singleton controlplane instead of cluster-manager. This is an alpha stage flag.")
	cmd.Flags().StringVar(&o.resourceQosClass, "resource-qos-class", "Default", "the resource QoS class of all the containers managed by the cluster manager and the cluster manager operator. Can be one of Default, BestEffort or ResourceRequirement.")
	cmd.Flags().StringToStringVar(&o.resourceLimits, "resource-limits", nil, "the resource limits of all the containers managed by the cluster manager and the cluster manager operator, for example: cpu=800m,memory=800Mi")
	cmd.Flags().StringToStringVar(&o.resourceRequests, "resource-requests", nil, "the resource requests of all the containers managed by the cluster manager and the cluster manager operator, for example: cpu=500m,memory=500Mi")
	cmd.Flags().BoolVar(&o.createNamespace, "create-namespace", true, "If true, create open-cluster-management namespace, otherwise use existing one")

	// clusterManagetSet contains the flags for deploy cluster-manager
	clusterManagerSet := pflag.NewFlagSet("clusterManagerSet", pflag.ExitOnError)
	cmd.Flags().StringVar(&o.registry, "image-registry", "quay.io/open-cluster-management",
		"The name of the image registry serving OCM images, which will be applied to all the deploying OCM components.")
	cmd.Flags().StringVar(&o.imagePullCredFile, "image-pull-credential-file", "",
		"The credential file is the docker config json file and will be filled into the default image pull secret named open-cluster-management-image-pull-credentials.")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")
	cmd.Flags().StringVar(&o.versionBundleFile, "bundle-version-overrides", "",
		"Path to a file containing version bundle overrides. Optional. If provided, overrides component versions within the selected version bundle.")
	clusterManagerSet.BoolVar(&o.useBootstrapToken, "use-bootstrap-token", false, "If set then the bootstrap token will used instead of a service account token")
	_ = clusterManagerSet.SetAnnotation("image-registry", "clusterManagerSet", []string{})
	_ = clusterManagerSet.SetAnnotation("bundle-version", "clusterManagerSet", []string{})
	_ = clusterManagerSet.SetAnnotation("bundle-version-file", "clusterManagerSet", []string{})
	_ = clusterManagerSet.SetAnnotation("use-bootstrap-token", "clusterManagerSet", []string{})
	cmd.Flags().AddFlagSet(clusterManagerSet)

	singletonSet := pflag.NewFlagSet("singletonSet", pflag.ExitOnError)
	singletonSet.StringVar(&o.SingletonName, "singleton-name", "singleton-controlplane", "The name of the singleton control plane")
	_ = clusterManagerSet.SetAnnotation("singleton-name", "singletonSet", []string{})
	o.Helm.AddFlags(singletonSet)
	cmd.Flags().AddFlagSet(singletonSet)
	cmd.Flags().StringSliceVar(&o.registrationDrivers, "registration-drivers", []string{},
		"The type of authentication to use for registering and authenticating with hub. Only csr and awsirsa are accepted as valid inputs. This flag can be repeated to specify multiple authentication types.")
	cmd.Flags().StringVar(&o.hubClusterArn, "hub-cluster-arn", "",
		"The hubCluster ARN to be passed if awsirsa is one of the registrationAuths and the cluster name in EKS kubeconfig doesn't contain hubClusterArn")
	cmd.Flags().StringSliceVar(&o.awsResourceTags, "aws-resource-tags", []string{},
		"List of tags to be added to AWS resources created by hub while processing awsirsa registration request, for example: product:v1:tenant:app-name=My-App,product:v1:tenant:created-by=Team-1")

	cmd.Flags().StringSliceVar(&o.autoApprovedCSRIdentities, "auto-approved-csr-identities", []string{},
		"The users or identities that can be auto approved for CSR and auto accepted to join with hub cluster")
	cmd.Flags().StringSliceVar(&o.autoApprovedARNPatterns, "auto-approved-arn-patterns", []string{},
		"List of AWS EKS ARN patterns so any EKS clusters with these patterns will be auto accepted to join with hub cluster")
	cmd.Flags().BoolVar(&o.enableSyncLabels, "enable-sync-labels", false, "If true, sync the labels from clustermanager to all hub resources.")

	return cmd
}
