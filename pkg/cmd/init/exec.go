// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/cli/values"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ocmfeature "open-cluster-management.io/api/feature"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/preflight"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
	clusteradmjson "open-cluster-management.io/clusteradm/pkg/helpers/json"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
	helperwait "open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

var (
	url         = "https://openclustermanagement.blob.core.windows.net/releases/"
	repoName    = "ocm"
	chartName   = "multicluster-controlplane"
	releaseName = "multicluster-controlplane"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun, "force", o.force, "output-file", o.outputFile)

	// ensure the flags are set correctly
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		fmt.Fprintf(o.Streams.Out, "flag %s has been set\n", flag.Name)
		if flag.Changed {
			_, ss := flag.Annotations["singletonSet"]
			_, cs := flag.Annotations["clusterManagerSet"]
			if !o.singleton && flag.Annotations != nil && ss {
				fmt.Fprintf(o.Streams.Out, "flag %s is only supported when deploy singleton controlplane\n", flag.Name)
			}
			if o.singleton && flag.Annotations != nil && cs {
				fmt.Fprintf(o.Streams.Out, "flag %s is only supported when deploy cluster manager\n", flag.Name)
			}
		}
	})

	if !o.singleton {
		o.values = Values{
			Hub: Hub{
				TokenID:     helpers.RandStringRunes_az09(6),
				TokenSecret: helpers.RandStringRunes_az09(16),
				Registry:    o.registry,
			},
			AutoApprove:          genericclioptionsclusteradm.HubMutableFeatureGate.Enabled(ocmfeature.ManagedClusterAutoApproval),
			RegistrationFeatures: genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubRegistrationFeatureGates),
			WorkFeatures:         genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubWorkFeatureGates),
			AddonFeatures:        genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubAddonManagerFeatureGates),
		}
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrieve version ", err)
		return err
	}

	o.values.BundleVersion = BundleVersion{
		RegistrationImageVersion: versionBundle.Registration,
		PlacementImageVersion:    versionBundle.Placement,
		WorkImageVersion:         versionBundle.Work,
		OperatorImageVersion:     versionBundle.Operator,
		AddonManagerImageVersion: versionBundle.AddonManager,
		ControlplaneImageVersion: versionBundle.MulticlusterControlplane,
	}

	f := o.ClusteradmFlags.KubectlFactory
	if !o.singleton {
		o.builder = f.NewBuilder()
	}

	return nil
}

func (o *Options) validate() error {
	if o.force {
		return nil
	}
	// preflight check
	f := o.ClusteradmFlags.KubectlFactory
	kubeClient, _, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}
	var checks []preflightinterface.Checker

	if o.singleton {
		checks = append(checks,
			preflight.SingletonControlplaneCheck{
				ControlplaneName: o.singletonValues.controlplaneName,
			})
	} else {
		checks = append(checks,
			preflight.HubApiServerCheck{
				ClusterCtx: o.ClusteradmFlags.Context,
				ConfigPath: "", // TODO(@Promacanthus)： user custom kubeconfig path from command line arguments.
			},
			preflight.ClusterInfoCheck{
				Namespace:    metav1.NamespacePublic,
				ResourceName: preflight.BootstrapConfigMap,
				ClusterCtx:   o.ClusteradmFlags.Context,
				ConfigPath:   "", // TODO(@Promacanthus)： user custom kubeconfig path from command line arguments.
				Client:       kubeClient,
			})
	}
	if err := preflightinterface.RunChecks(checks, os.Stderr); err != nil {
		return err
	}

	if len(o.registry) == 0 {
		return fmt.Errorf("registry should not be empty")
	}

	// If --wait is set, some information during initialize process will print to output, the output would not keep
	// machine readable, so this behavior should be disabled
	if o.wait && o.output != "text" {
		return fmt.Errorf("output should be text if --wait is set")
	}
	return nil
}

func (o *Options) run() error {
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	if o.singleton {
		err = o.deploySingletonControlplane(kubeClient)
		if err != nil {
			return err
		}
	} else {
		token := fmt.Sprintf("%s.%s", o.values.Hub.TokenID, o.values.Hub.TokenSecret)

		files := []string{
			"init/namespace.yaml",
		}
		if o.useBootstrapToken {
			files = append(files,
				"init/bootstrap-token-secret.yaml",
				"init/bootstrap_cluster_role.yaml",
				"init/bootstrap_cluster_role_binding.yaml",
			)
		} else {
			files = append(files,
				"init/bootstrap_sa.yaml",
				"init/bootstrap_cluster_role.yaml",
				"init/bootstrap_sa_cluster_role_binding.yaml",
			)
		}

		files = append(files,
			"init/clustermanager_cluster_role.yaml",
			"init/clustermanager_cluster_role_binding.yaml",
			"init/clustermanagers.crd.yaml",
			"init/clustermanager_sa.yaml",
		)

		r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)
		err = r.Apply(scenario.Files, o.values, files...)
		if err != nil {
			return err
		}

		err = r.Apply(scenario.Files, o.values, "init/operator.yaml")
		if err != nil {
			return err
		}

		if !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilCRDReady(apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", o.wait); err != nil {
				return err
			}
		}
		if o.wait && !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilRegistrationOperatorReady(
				o.ClusteradmFlags.KubectlFactory,
				int64(o.ClusteradmFlags.Timeout)); err != nil {
				return err
			}
		}

		err = r.Apply(scenario.Files, o.values, "init/clustermanager.cr.yaml")
		if err != nil {
			return err
		}

		if o.wait && !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilClusterManagerRegistrationReady(
				o.ClusteradmFlags.KubectlFactory,
				int64(o.ClusteradmFlags.Timeout)); err != nil {
				return err
			}
		}

		//if service-account wait for the sa secret
		if !o.useBootstrapToken && !o.ClusteradmFlags.DryRun {
			token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
			if err != nil {
				return err
			}
		}

		restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
		if err != nil {
			return nil
		}

		cmd := fmt.Sprintf("%s join --hub-token %s --hub-apiserver %s",
			helpers.GetExampleHeader(),
			token,
			restConfig.Host)

		// if the init command prescribes a foreground installation, adds the `--wait`
		// flag to the join command to cohere the behavior of init and join commands.
		if o.wait {
			cmd = cmd + " --wait"
		}

		if len(o.outputJoinCommandFile) > 0 {
			sh, err := os.OpenFile(o.outputJoinCommandFile, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(sh, "%s --cluster-name $1", cmd)
			if err != nil {
				return err
			}
			if err := sh.Close(); err != nil {
				return err
			}
		}

		if len(o.outputFile) > 0 {
			sh, err := os.OpenFile(o.outputFile, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(sh, "%s", string(r.RawAppliedResources()))
			if err != nil {
				return err
			}
			if err := sh.Close(); err != nil {
				return err
			}
		}

		if o.output == "json" {
			err := clusteradmjson.WriteJsonOutput(o.Streams.Out, clusteradmjson.HubInfo{
				HubToken:     token,
				HubApiserver: restConfig.Host,
			})
			if err != nil {
				return err
			}
		} else {
			fmt.Fprintf(o.Streams.Out, "The multicluster hub control plane has been initialized successfully!\n\n"+
				"You can now register cluster(s) to the hub control plane. Log onto those cluster(s) and run the following command:\n\n"+
				"    %s --cluster-name <cluster_name>\n\n"+
				"Replace <cluster_name> with a cluster name of your choice. For example, cluster1.\n\n",
				cmd,
			)
		}
	}
	return nil
}

func (o *Options) deploySingletonControlplane(kubeClient kubernetes.Interface) error {
	// create namespace
	_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), o.singletonValues.controlplaneName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: o.singletonValues.controlplaneName,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	h := helm.NewHelmWithNamespace(o.singletonValues.controlplaneName)
	err = h.PrepareChart(repoName, url)
	if err != nil {
		return err
	}

	// set values
	valueOpts := &values.Options{
		Values:     []string{},
		FileValues: []string{},
	}

	if o.registry != "quay.io/open-cluster-management" {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("image=%s", o.registry+"/multicluster-controlplane:"+o.values.BundleVersion.ControlplaneImageVersion))
	}

	if o.singletonValues.autoApprovalBootstrapUsers != "" {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("autoApprovalBootstrapUsers=%s", o.singletonValues.autoApprovalBootstrapUsers))
	}
	if o.singletonValues.enableSelfManagement {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("enableSelfManagement=%t", o.singletonValues.enableSelfManagement))
	}
	if o.singletonValues.enableDelegatingAuthentication {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("enableDelegatingAuthentication=%t", o.singletonValues.enableDelegatingAuthentication))
	}

	if o.singletonValues.apiserverExternalHostname != "" {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("apiserver.externalHostname=%s", o.singletonValues.apiserverExternalHostname))
	}
	if o.singletonValues.apiserverCA != "" && o.singletonValues.apiserverCAKey != "" {
		valueOpts.FileValues = append(valueOpts.FileValues, fmt.Sprintf("apiserver.ca=%s", o.singletonValues.apiserverCA))
		valueOpts.FileValues = append(valueOpts.FileValues, fmt.Sprintf("apiserver.cakey=%s", o.singletonValues.apiserverCAKey))
	}

	if o.singletonValues.etcdMode == "embed" || o.singletonValues.etcdMode == "external" {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("etcd.mode=%s", o.singletonValues.etcdMode))
	}
	if o.singletonValues.etcdServers != nil {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("etcd.servers={%s}", strings.Join(o.singletonValues.etcdServers, ",")))
	}
	if o.singletonValues.etcdCA != "" && o.singletonValues.etcdClientCert != "" && o.singletonValues.etcdClientCertKey != "" {
		valueOpts.FileValues = append(valueOpts.FileValues, fmt.Sprintf("etcd.ca=%s", o.singletonValues.etcdCA))
		valueOpts.FileValues = append(valueOpts.FileValues, fmt.Sprintf("etcd.cert=%s", o.singletonValues.etcdClientCert))
		valueOpts.FileValues = append(valueOpts.FileValues, fmt.Sprintf("etcd.certkey=%s", o.singletonValues.etcdClientCertKey))
	}

	if o.singletonValues.pvcStorageClassName != "" {
		valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("pvc.storageClassName=%s", o.singletonValues.pvcStorageClassName))
	}

	if o.singletonValues.routeEnabled {
		valueOpts.Values = append(valueOpts.Values, "route.enabled=true")
	} else {
		valueOpts.Values = append(valueOpts.Values, "route.enabled=false")
		if o.singletonValues.loadBalancerEnabled {
			valueOpts.Values = append(valueOpts.Values, "loadbalancer.enabled=true")
			if o.singletonValues.loadBalancerBaseDomain != "" {
				valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("loadbalancer.baseDomain=%s", o.singletonValues.loadBalancerBaseDomain))
			}
		} else if o.singletonValues.nodeportEnabled {
			valueOpts.Values = append(valueOpts.Values, "nodeport.enabled=true")
			if o.singletonValues.nodeportValue != 0 {
				valueOpts.Values = append(valueOpts.Values, fmt.Sprintf("nodeport.port=%d", o.singletonValues.nodeportValue))
			}
		}
	}

	if o.ClusteradmFlags.DryRun {
		valueOpts.Values = append(valueOpts.Values, "dryRun=true")
	}
	h.InstallChart(releaseName, repoName, chartName, valueOpts)

	// fetch the kubeconfig and get the token
	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := helperwait.WaitUntilMulticlusterControlplaneReady(
			o.ClusteradmFlags.KubectlFactory,
			o.singletonValues.controlplaneName,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}

		b := retry.DefaultBackoff
		b.Duration = 3 * time.Second
		if err := helperwait.WaitUntilMulticlusterControlplaneKubeconfigReady(
			o.ClusteradmFlags.KubectlFactory,
			o.singletonValues.controlplaneName,
			b); err != nil {
			return err
		}

		// if kubeconfig is ready, get kubeconfig from secret, write to file or outpout to stdout
		conf, err := kubeClient.CoreV1().Secrets(o.singletonValues.controlplaneName).Get(context.Background(), "multicluster-controlplane-kubeconfig", metav1.GetOptions{})
		if err != nil {
			return err
		}
		kubeconfigRaw := conf.Data["kubeconfig"]
		kubeconfigfile := fmt.Sprintf("%s.kubeconfig", o.singletonValues.controlplaneName)
		if err := os.WriteFile(kubeconfigfile, kubeconfigRaw, 0600); err != nil {
			return err
		}

		fmt.Fprintf(o.Streams.Out, "The multicluster controlplane has been initialized successfully!\n"+
			"You can use "+fmt.Sprintf("\"kubectl --kubeconfig %s\"", kubeconfigfile)+" to access control plane.\n\n")
	}
	return nil
}
