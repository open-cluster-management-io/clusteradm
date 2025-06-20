// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/preflight"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/helm"
	clusteradmjson "open-cluster-management.io/clusteradm/pkg/helpers/json"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/resourcerequirement"
	helperwait "open-cluster-management.io/clusteradm/pkg/helpers/wait"
	"open-cluster-management.io/clusteradm/pkg/version"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

var (
	url         = "https://open-cluster-management.io/helm-charts"
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
			_, hs := flag.Annotations[helm.HelmFlagSetAnnotation]
			_, ss := flag.Annotations["singletonSet"]
			_, cs := flag.Annotations["clusterManagerSet"]
			// Falgs in helm set or singleton set can be set together
			if !o.singleton && flag.Annotations != nil && (ss || hs) {
				fmt.Fprintf(o.Streams.Out, "flag %s is only supported when deploy singleton controlplane\n", flag.Name)
			}
			if o.singleton && flag.Annotations != nil && cs {
				fmt.Fprintf(o.Streams.Out, "flag %s is only supported when deploy cluster manager\n", flag.Name)
			}
		}
	})

	bundleVersion, err := version.GetVersionBundle(o.bundleVersion, o.versionBundleFile)
	if err != nil {
		return err
	}

	o.clusterManagerChartConfig.EnableSyncLabels = o.enableSyncLabels

	if !o.singleton {
		o.clusterManagerChartConfig.Images = chart.ImagesConfig{
			Registry: o.registry,
			ImageCredentials: chart.ImageCredentials{
				CreateImageCredentials: true,
			},
			Tag: bundleVersion.OCM,
		}
		registrationDrivers, err := getRegistrationDrivers(o)
		if err != nil {
			return err
		}

		o.clusterManagerChartConfig.ClusterManager = chart.ClusterManagerConfig{
			RegistrationConfiguration: operatorv1.RegistrationHubConfiguration{
				FeatureGates: genericclioptionsclusteradm.ConvertToFeatureGateAPI(
					genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubRegistrationFeatureGates),
				RegistrationDrivers: registrationDrivers,
			},
			WorkConfiguration: operatorv1.WorkConfiguration{
				FeatureGates: genericclioptionsclusteradm.ConvertToFeatureGateAPI(
					genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubWorkFeatureGates),
			},
			AddOnManagerConfiguration: operatorv1.AddOnManagerConfiguration{
				FeatureGates: genericclioptionsclusteradm.ConvertToFeatureGateAPI(
					genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubAddonManagerFeatureGates),
			},
		}
		o.clusterManagerChartConfig.CreateBootstrapToken = o.useBootstrapToken

		if o.imagePullCredFile != "" {
			content, err := os.ReadFile(o.imagePullCredFile)
			if err != nil {
				return fmt.Errorf("failed read the image pull credential file %v: %v", o.imagePullCredFile, err)
			}
			o.clusterManagerChartConfig.Images.ImageCredentials.DockerConfigJson = string(content)
		}

		resourceRequirement, err := resourcerequirement.NewResourceRequirement(
			operatorv1.ResourceQosClass(o.resourceQosClass), o.resourceLimits, o.resourceRequests)
		if err != nil {
			return err
		}
		o.clusterManagerChartConfig.ClusterManager.ResourceRequirement = *resourceRequirement
	} else {
		o.Helm.WithNamespace(o.SingletonName)
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
				ControlplaneName: o.SingletonName,
			})
	} else {
		checks = append(checks,
			preflight.HubApiServerCheck{
				Config: o.ClusteradmFlags.KubectlFactory.ToRawKubeConfigLoader(),
			},
			preflight.ClusterInfoCheck{
				Namespace:    metav1.NamespacePublic,
				ResourceName: preflight.BootstrapConfigMap,
				Config:       o.ClusteradmFlags.KubectlFactory.ToRawKubeConfigLoader(),
				Client:       kubeClient,
			})
	}
	if err := preflightinterface.RunChecks(checks, os.Stderr); err != nil {
		return err
	}

	if len(o.registry) == 0 {
		return fmt.Errorf("registry should not be empty")
	}

	validRegistrationDriver := sets.New[string]("csr", "awsirsa")
	for _, driver := range o.registrationDrivers {
		if !validRegistrationDriver.Has(driver) {
			return fmt.Errorf("only csr and awsirsa are valid drivers")
		}
	}

	if genericclioptionsclusteradm.HubMutableFeatureGate.Enabled("ManagedClusterAutoApproval") {
		// If hub registration does not accept awsirsa, we stop user if they also pass in a list of patterns for AWS EKS ARN.

		if len(o.autoApprovedARNPatterns) > 0 && !sets.New[string](o.registrationDrivers...).Has("awsirsa") {
			return fmt.Errorf("should not provide list of patterns for aws eks arn if not initializing hub with awsirsa registration")
		}

		// If hub registration does not accept csr, we stop user if they also pass in a list of users for CSR auto approval.
		if len(o.autoApprovedCSRIdentities) > 0 && !sets.New[string](o.registrationDrivers...).Has("csr") {
			return fmt.Errorf("should not provide list of users for csr to auto approve if not initializing hub with csr registration")
		}
	} else if len(o.autoApprovedARNPatterns) > 0 || len(o.autoApprovedCSRIdentities) > 0 {
		return fmt.Errorf("should enable feature gate ManagedClusterAutoApproval before passing list of identities")
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
		o.clusterManagerChartConfig.CreateNamespace = o.createNamespace
		if !o.createNamespace {
			fmt.Fprintf(o.Streams.Out, "skip creating namespace\n")
		}

		if !o.useBootstrapToken {
			o.clusterManagerChartConfig.CreateBootstrapSA = true
		} else {
			o.clusterManagerChartConfig.CreateBootstrapToken = true
		}

		r := reader.NewResourceReader(o.ClusteradmFlags.KubectlFactory, o.ClusteradmFlags.DryRun, o.Streams)
		crds, raw, err := chart.RenderClusterManagerChart(
			o.clusterManagerChartConfig,
			"open-cluster-management")
		if err != nil {
			return err
		}

		if err := r.ApplyRaw(crds); err != nil {
			return err
		}

		if !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilCRDReady(
				o.Streams.Out, apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", o.wait); err != nil {
				return err
			}
		}

		if err := r.ApplyRaw(raw); err != nil {
			return err
		}

		if o.wait && !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilRegistrationOperatorReady(
				o.Streams.Out,
				o.ClusteradmFlags.KubectlFactory,
				int64(o.ClusteradmFlags.Timeout)); err != nil {
				return err
			}
		}

		if o.wait && !o.ClusteradmFlags.DryRun {
			if err := helperwait.WaitUntilClusterManagerRegistrationReady(
				o.Streams.Out,
				o.ClusteradmFlags.KubectlFactory,
				int64(o.ClusteradmFlags.Timeout)); err != nil {
				return err
			}
		}

		// if service-account wait for the sa secret
		var token string
		if !o.useBootstrapToken && !o.ClusteradmFlags.DryRun {
			token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
			if err != nil {
				return err
			}
		} else if !o.ClusteradmFlags.DryRun {
			token, err = helpers.GetBootstrapToken(context.TODO(), kubeClient)
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
	_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), o.SingletonName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: o.SingletonName,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	err = o.Helm.PrepareChart(repoName, url)
	if err != nil {
		return err
	}

	if o.ClusteradmFlags.DryRun {
		o.Helm.SetValue("dryRun", "true")
	}

	o.Helm.InstallChart(releaseName, repoName, chartName)

	// fetch the kubeconfig and get the token
	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := helperwait.WaitUntilMulticlusterControlplaneReady(
			o.Streams.Out,
			o.ClusteradmFlags.KubectlFactory,
			o.SingletonName,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}

		b := retry.DefaultBackoff
		b.Duration = 3 * time.Second
		if err := helperwait.WaitUntilMulticlusterControlplaneKubeconfigReady(
			o.ClusteradmFlags.KubectlFactory,
			o.SingletonName,
			b); err != nil {
			return err
		}

		// if kubeconfig is ready, get kubeconfig from secret, write to file or outpout to stdout
		conf, err := kubeClient.CoreV1().Secrets(o.SingletonName).Get(context.Background(), "multicluster-controlplane-kubeconfig", metav1.GetOptions{})
		if err != nil {
			return err
		}
		kubeconfigRaw := conf.Data["kubeconfig"]
		kubeconfigfile := fmt.Sprintf("%s.kubeconfig", o.SingletonName)
		if err := os.WriteFile(kubeconfigfile, kubeconfigRaw, 0600); err != nil {
			return err
		}

		fmt.Fprintf(o.Streams.Out, "The multicluster controlplane has been initialized successfully!\n"+
			"You can use "+fmt.Sprintf("\"kubectl --kubeconfig %s\"", kubeconfigfile)+" to access control plane.\n\n")
	}
	return nil
}

func getRegistrationDrivers(o *Options) ([]operatorv1.RegistrationDriverHub, error) {
	registrationDrivers := []operatorv1.RegistrationDriverHub{}
	var registrationDriver operatorv1.RegistrationDriverHub

	for _, driver := range o.registrationDrivers {
		if driver == "csr" {
			csr := &operatorv1.CSRConfig{AutoApprovedIdentities: o.autoApprovedCSRIdentities}
			registrationDriver = operatorv1.RegistrationDriverHub{AuthType: driver, CSR: csr}
		} else if driver == "awsirsa" {
			hubClusterArn, err := getHubClusterArn(o)
			if err != nil {
				return registrationDrivers, err
			}
			awsirsa := &operatorv1.AwsIrsaConfig{HubClusterArn: hubClusterArn, Tags: o.awsResourceTags, AutoApprovedIdentities: o.autoApprovedARNPatterns}
			registrationDriver = operatorv1.RegistrationDriverHub{AuthType: driver, AwsIrsa: awsirsa}
		}
		registrationDrivers = append(registrationDrivers, registrationDriver)
	}
	return registrationDrivers, nil
}

func getHubClusterArn(o *Options) (string, error) {
	hubClusterArn := o.hubClusterArn
	if hubClusterArn == "" {
		rawConfig, err := o.ClusteradmFlags.KubectlFactory.ToRawKubeConfigLoader().RawConfig()
		if err != nil {
			klog.Errorf("unable to load hub cluster kubeconfig: %v", err)
			return "", err
		}
		hubClusterArn = rawConfig.Contexts[rawConfig.CurrentContext].Cluster
		if hubClusterArn == "" {
			klog.Errorf("hubClusterArn has empty value in kubeconfig")
			return "", fmt.Errorf("unable to retrieve hubClusterArn from kubeconfig")
		}
	}
	return hubClusterArn, nil
}
