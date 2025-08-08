// Copyright Contributors to the Open Cluster Management project
package join

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

var example = `
# Join a cluster to the hub 
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name>
# join a cluster to the hub with hosted mode
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --mode hosted --managed-cluster-kubeconfig <managed-cluster-kubeconfig-file>
# join a cluster to the hub while the hub provided no valid CA data in kube-public namespace
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --ca-file <ca-file>
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --registration-auth awsirsa --hub-cluster-arn arn:aws:eks:us-west-2:123456789012:cluster/hub-cluster-1
# Join a cluster to the hub and annotate the ManagedCluster
%[1]s join --hub-token <tokenID.tokenSecret> --hub-apiserver <hub_apiserver_url> --cluster-name <cluster_name> --klusterlet-annotation foo=bar --klusterlet-annotation bar=foo
`

// NewCmd ...
func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericiooptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags, streams)

	cmd := &cobra.Command{
		Use:          "join",
		Short:        "join a cluster to the hub",
		Long:         "join specific cluster to the hub cluster",
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

	genericclioptionsclusteradm.SpokeMutableFeatureGate.AddFlag(cmd.Flags())
	o.capiOptions.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&o.token, "hub-token", "", "The token to access the hub")
	cmd.Flags().StringVar(&o.hubAPIServer, "hub-apiserver", "", "The api server url to the hub")
	cmd.Flags().StringVar(&o.caFile, "ca-file", "", "the file path to hub ca, optional")
	cmd.Flags().StringVar(&o.clusterName, "cluster-name", "", "The name of the joining cluster")
	cmd.Flags().StringVar(&o.outputFile, "output-file", "", "The generated resources will be copied in the specified file")
	cmd.Flags().StringVar(&o.registry, "image-registry", "quay.io/open-cluster-management", "The name of the image registry serving OCM images.")
	cmd.Flags().StringVar(&o.imagePullCredFile, "image-pull-credential-file", "",
		"The credential file is the docker config json file and will be filled into the default image pull secret named open-cluster-management-image-pull-credentials.")
	cmd.Flags().StringVar(&o.bundleVersion, "bundle-version", "default",
		"The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version. You can also set \"latest\" to install the latest development version.")
	cmd.Flags().StringVar(&o.versionBundleFile, "bundle-version-overrides", "",
		"Path to a file containing version bundle overrides. Optional. If provided, overrides component versions within the selected version bundle.")
	cmd.Flags().BoolVar(&o.forceHubInClusterEndpointLookup, "force-internal-endpoint-lookup", false,
		"If true, the installed klusterlet agent will be starting the cluster registration process by "+
			"looking for the internal endpoint from the public cluster-info in the hub cluster instead of from --hub-apiserver.")
	cmd.Flags().BoolVar(&o.forceManagedInClusterEndpointLookup, "force-internal-endpoint-lookup-managed", false,
		"If true, the klusterlet accesses the managed cluster by using the internal endpoint from the public cluster-info"+
			" in the managed cluster instead of from --managed-cluster-kubeconfig directly.")
	cmd.Flags().BoolVar(&o.wait, "wait", false, "If true, running the cluster registration in foreground.")
	cmd.Flags().StringVarP(&o.mode, "mode", "m", "default", "mode to deploy klusterlet, can be default or hosted")
	cmd.Flags().StringVar(&o.managedKubeconfigFile, "managed-cluster-kubeconfig", "", "To specify the directory to external managed cluster kubeconfig in hosted mode")
	cmd.Flags().BoolVar(&o.singleton, "singleton", false, "If true, deploy singleton mode of klusterlet to have registration and work agents run in a single pod. This is an alpha stage flag.")
	cmd.Flags().StringVar(&o.proxyURL, "proxy-url", "", "the URL of a forward proxy server that will be used by agents to connect to the hub cluster.")
	cmd.Flags().StringVar(&o.proxyCAFile, "proxy-ca-file", "", "the file path to proxy ca, optional")
	cmd.Flags().StringVar(&o.resourceQosClass, "resource-qos-class", "Default", "the resource QoS class of all the containers managed by the klusterlet and the klusterlet operator. Can be one of Default, BestEffort or ResourceRequirement.")
	cmd.Flags().StringToStringVar(&o.resourceLimits, "resource-limits", nil, "the resource limits of all the containers managed by the klusterlet and the klusterlet operator, for example: cpu=800m,memory=800Mi")
	cmd.Flags().StringToStringVar(&o.resourceRequests, "resource-requests", nil, "the resource requests of all the containers managed by the klusterlet and the klusterlet operator, for example: cpu=500m,memory=500Mi")
	cmd.Flags().BoolVar(&o.createNameSpace, "create-namespace", true, "If true, create the operator namespace(open-cluster-management) and the agent namespace(open-cluster-management-agent for Default mode, <klusterlet-name> for Hosted mode), otherwise use existing one")
	cmd.Flags().BoolVar(&o.enableSyncLabels, "enable-sync-labels", false, "If true, sync the labels from klusterlet to all agent resources.")
	cmd.Flags().Int32Var(&o.clientCertExpirationSeconds, "client-cert-expiration-seconds", 31536000, "clientCertExpirationSeconds represents the seconds of a client certificate to expire.")
	cmd.Flags().StringVar(&o.registrationAuth, "registration-auth", "", "The type of authentication to use for registering and authenticating with hub")
	cmd.Flags().StringVar(&o.hubClusterArn, "hub-cluster-arn", "", "The arn of the hub cluster(i.e. EKS cluster) to which managed-cluster will join")
	cmd.Flags().StringVar(&o.managedClusterArn, "managed-cluster-arn", "", "The arn of the managed cluster(i.e. EKS cluster) which will be joining the hub")
	cmd.Flags().StringArrayVar(&o.klusterletAnnotations, "klusterlet-annotation", []string{}, fmt.Sprintf("Annotations to set on the ManagedCluster, in key=value format. Note: each key will be automatically prefixed with '%s/', if not set.", operatorv1.ClusterAnnotationsKeyPrefix))
	cmd.Flags().StringVar(&o.klusterletValuesFile, "klusterlet-values-file", "", "The path to a YAML file containing klusterlet Helm chart values. The values from the file override the default klusterlet chart values.")
	return cmd
}
