// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	ocmfeature "open-cluster-management.io/api/feature"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/preflight"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

const (
	AgentNamespacePrefix = "open-cluster-management-"

	InstallModeDefault = "Default"
	InstallModeHosted  = "Hosted"

	OperatorNamesapce   = "open-cluster-management"
	DefaultOperatorName = "klusterlet"
)

func format(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if cmd.Flags() == nil {
		return fmt.Errorf("no flags have been set: hub-apiserver, hub-token and cluster-name is required")
	}

	if o.token == "" {
		return fmt.Errorf("token is missing")
	}
	if o.hubAPIServer == "" {
		return fmt.Errorf("hub-server is missing")
	}
	if o.clusterName == "" {
		return fmt.Errorf("cluster-name is missing")
	}
	if len(o.registry) == 0 {
		return fmt.Errorf("the OCM image registry should not be empty, like quay.io/open-cluster-management")
	}

	if len(o.mode) == 0 {
		return fmt.Errorf("the mode should not be empty, like default")
	}
	// convert mode string to lower
	o.mode = format(o.mode)

	klog.V(1).InfoS("join options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, "api-server", o.hubAPIServer, "output", o.outputFile)

	agentNamespace := AgentNamespacePrefix + "agent"

	o.values = Values{
		ClusterName: o.clusterName,
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
		Registry:       o.registry,
		AgentNamespace: agentNamespace,
	}

	if o.singleton { // deploy singleton agent
		if o.mode != InstallModeDefault {
			return fmt.Errorf("only default mode is supported while deploy singleton agent, hosted mode will be supported in the future")
		}
	} else { // deploy klusterlet
		// operatorNamespace is the namespace to deploy klsuterlet;
		// agentNamespace is the namesapce to deploy the agents(registration agent, work agent, etc.);
		// klusterletNamespace is the namespace created on the managed cluster for each klusterlet.
		//
		// The operatorNamespace is fixed to "open-cluster-management".
		// In default mode, agentNamespace is "open-cluster-management-agent", klusterletNamespace refers to agentNamespace, all of these three namesapces are on the managed cluster;
		// In hosted mode, operatorNamespace is on the management cluster, agentNamesapce is "<cluster name>-<6-bit random string>" on the management cluster, and the klusterletNamespace is "open-cluster-management-<agentNamespace>" on the managed cluster.

		// values for default mode
		klusterletName := DefaultOperatorName
		klusterletNamespace := agentNamespace
		if o.mode == InstallModeHosted {
			// add hash suffix to avoid conflict
			klusterletName += "-hosted-" + helpers.RandStringRunes_az09(6)
			agentNamespace = klusterletName
			klusterletNamespace = AgentNamespacePrefix + agentNamespace

			// update AgentNamespace
			o.values.AgentNamespace = agentNamespace
		}

		o.values.Klusterlet = Klusterlet{
			Mode:                o.mode,
			Name:                klusterletName,
			KlusterletNamespace: klusterletNamespace,
		}
		o.values.ManagedKubeconfig = o.managedKubeconfigFile
		o.values.RegistrationFeatures = genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeRegistrationFeatureGates)
		o.values.WorkFeatures = genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeWorkFeatureGates)
	}
	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrieve version ", err)
		return err
	}

	o.values.BundleVersion = BundleVersion{
		RegistrationImageVersion:   versionBundle.Registration,
		PlacementImageVersion:      versionBundle.Placement,
		WorkImageVersion:           versionBundle.Work,
		OperatorImageVersion:       versionBundle.Operator,
		SingletonAgentImageVersion: versionBundle.MulticlusterControlplane,
	}
	klog.V(3).InfoS("Image version:",
		"'registration image version'", versionBundle.Registration,
		"'placement image version'", versionBundle.Placement,
		"'work image version'", versionBundle.Work,
		"'operator image version'", versionBundle.Operator,
		"'singleton agent image version'", versionBundle.MulticlusterControlplane)

	// if --ca-file is set, read ca data
	if o.caFile != "" {
		cabytes, err := os.ReadFile(o.caFile)
		if err != nil {
			return err
		}
		o.HubCADate = cabytes
	}

	// code logic of building hub client in join process:
	// 1. use the token and insecure to fetch the ca data from cm in kube-public ns
	// 2. if not found, assume using an authorized ca.
	// 3. use the ca and token to build a secured client and call hub

	//Create an unsecure bootstrap
	bootstrapExternalConfigUnSecure := o.createExternalBootstrapConfig()
	//create external client from the bootstrap
	externalClientUnSecure, err := helpers.CreateClientFromClientcmdapiv1Config(bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}
	//Create the kubeconfig for the internal client
	o.HubConfig, err = o.createClientcmdapiv1Config(externalClientUnSecure, bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}

	// get managed cluster externalServerURL
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		klog.Errorf("Failed building kube client: %v", err)
		return err
	}
	klusterletApiserver, err := helpers.GetAPIServer(kubeClient)
	if err != nil {
		klog.Warningf("Failed looking for cluster endpoint for the registering klusterlet: %v", err)
		klusterletApiserver = ""
	} else if !preflight.ValidAPIHost(klusterletApiserver) {
		klog.Warningf("ConfigMap/cluster-info.data.kubeconfig.clusters[0].cluster.server field [%s] in namespace kube-public should start with http:// or https://", klusterletApiserver)
		klusterletApiserver = ""
	}
	o.values.Klusterlet.APIServer = klusterletApiserver

	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()

	klog.V(3).InfoS("values:",
		"clusterName", o.values.ClusterName,
		"hubAPIServer", o.values.Hub.APIServer,
		"klusterletAPIServer", o.values.Klusterlet.APIServer)
	return nil

}

func (o *Options) validate() error {
	// preflight check
	if err := preflightinterface.RunChecks(
		[]preflightinterface.Checker{
			preflight.HubKubeconfigCheck{
				Config: o.HubConfig,
			},
			preflight.DeployModeCheck{
				Mode:                  o.mode,
				InternalEndpoint:      o.forceHubInClusterEndpointLookup,
				ManagedKubeconfigFile: o.managedKubeconfigFile,
			},
			preflight.ClusterNameCheck{
				ClusterName: o.values.ClusterName,
			},
		}, os.Stderr); err != nil {
		return err
	}

	err := o.setKubeconfig()
	if err != nil {
		return err
	}

	// get ManagedKubeconfig from given file
	if o.mode == InstallModeHosted {
		managedConfig, err := os.ReadFile(o.managedKubeconfigFile)
		if err != nil {
			return err
		}
		o.values.ManagedKubeconfig = base64.StdEncoding.EncodeToString(managedConfig)
	}

	return nil
}

func (o *Options) run() error {
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)

	_, err = kubeClient.CoreV1().Namespaces().Get(context.TODO(), o.values.AgentNamespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: o.values.AgentNamespace,
					Annotations: map[string]string{
						"workload.openshift.io/allowed": "management",
					},
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if o.singleton {
		err = o.applySingletonAgent(r, kubeClient)
		if err != nil {
			return err
		}
	} else {
		err = o.applyKlusterlet(r, kubeClient, apiExtensionsClient)
		if err != nil {
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

	fmt.Fprintf(o.Streams.Out, "Please log onto the hub cluster and run the following command:\n\n"+
		"    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), o.values.ClusterName)
	return nil

}

func (o *Options) applySingletonAgent(r *reader.ResourceReader, kubeClient kubernetes.Interface) error {
	files := []string{
		"bootstrap_hub_kubeconfig.yaml",
		"singleton/clusterrole.yaml",
		"singleton/clusterrolebinding-admin.yaml",
		"singleton/clusterrolebinding.yaml",
		"singleton/role.yaml",
		"singleton/rolebinding.yaml",
		"singleton/serviceaccount.yaml",
		"singleton/deployment.yaml",
	}

	err := r.Apply(scenario.Files, o.values, files...)
	if err != nil {
		return err
	}

	if o.wait && !o.ClusteradmFlags.DryRun {
		err = waitUntilSingletonAgentConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout), o.values.AgentNamespace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) applyKlusterlet(r *reader.ResourceReader, kubeClient kubernetes.Interface, apiExtensionsClient apiextensionsclient.Interface) error {
	available, err := checkIfRegistrationOperatorAvailable(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	files := []string{}
	// If Deployment/klusterlet is not deployed, deploy it
	if !available {
		files = append(files,
			"join/klusterlets.crd.yaml",
			"join/namespace.yaml",
			"join/service_account.yaml",
			"join/cluster_role.yaml",
			"join/cluster_role_binding.yaml",
		)
	}
	files = append(files,
		"bootstrap_hub_kubeconfig.yaml",
	)

	if o.mode == InstallModeHosted {
		files = append(files,
			"join/hosted/external_managed_kubeconfig.yaml",
		)
	}

	err = r.Apply(scenario.Files, o.values, files...)
	if err != nil {
		return err
	}

	if !available {
		err = r.Apply(scenario.Files, o.values, "join/operator.yaml")
		if err != nil {
			return err
		}
	}

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}

	err = r.Apply(scenario.Files, o.values, "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}

	klusterletNamespace := o.values.Klusterlet.KlusterletNamespace
	agentNamespace := o.values.AgentNamespace

	if !available && o.wait && !o.ClusteradmFlags.DryRun {
		err = waitUntilRegistrationOperatorConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout))
		if err != nil {
			return err
		}
	}

	if o.wait && !o.ClusteradmFlags.DryRun {
		if o.mode == InstallModeHosted {
			err = waitUntilKlusterletConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout), agentNamespace)
			if err != nil {
				return err
			}
		} else {
			err = waitUntilKlusterletConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout), klusterletNamespace)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkIfRegistrationOperatorAvailable(f util.Factory) (bool, error) {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return false, err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	deploy, err := client.AppsV1().Deployments(OperatorNamesapce).
		Get(context.TODO(), DefaultOperatorName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	conds := make([]metav1.Condition, len(deploy.Status.Conditions))
	for i := range deploy.Status.Conditions {
		conds[i] = metav1.Condition{
			Type:    string(deploy.Status.Conditions[i].Type),
			Status:  metav1.ConditionStatus(deploy.Status.Conditions[i].Status),
			Reason:  deploy.Status.Conditions[i].Reason,
			Message: deploy.Status.Conditions[i].Message,
		}
	}
	return meta.IsStatusConditionTrue(conds, "Available"), nil
}

func waitUntilRegistrationOperatorConditionIsTrue(f util.Factory, timeout int64) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	operatorSpinner := printer.NewSpinnerWithStatus(
		"Waiting for registration operator to become ready...",
		time.Millisecond*500,
		"Registration operator is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	operatorSpinner.Start()
	defer operatorSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods(OperatorNamesapce).
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					LabelSelector:  "app=klusterlet",
				})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
			conds := make([]metav1.Condition, len(pod.Status.Conditions))
			for i := range pod.Status.Conditions {
				conds[i] = metav1.Condition{
					Type:    string(pod.Status.Conditions[i].Type),
					Status:  metav1.ConditionStatus(pod.Status.Conditions[i].Status),
					Reason:  pod.Status.Conditions[i].Reason,
					Message: pod.Status.Conditions[i].Message,
				}
			}
			return meta.IsStatusConditionTrue(conds, "Ready")
		})
}

// Wait until the klusterlet condition available=true, or timeout in $timeout seconds
func waitUntilKlusterletConditionIsTrue(f util.Factory, timeout int64, agentNamespace string) error {
	client, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	klusterletSpinner := printer.NewSpinnerWithStatus(
		"Waiting for klusterlet agent to become ready...",
		time.Millisecond*500,
		"Klusterlet is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	klusterletSpinner.Start()
	defer klusterletSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods(agentNamespace).
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					LabelSelector:  "app=klusterlet-registration-agent",
				})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
			conds := make([]metav1.Condition, len(pod.Status.Conditions))
			for i := range pod.Status.Conditions {
				conds[i] = metav1.Condition{
					Type:    string(pod.Status.Conditions[i].Type),
					Status:  metav1.ConditionStatus(pod.Status.Conditions[i].Status),
					Reason:  pod.Status.Conditions[i].Reason,
					Message: pod.Status.Conditions[i].Message,
				}
			}
			return meta.IsStatusConditionTrue(conds, "Ready")
		},
	)
}

func waitUntilSingletonAgentConditionIsTrue(f util.Factory, timeout int64, agentNamespace string) error {
	client, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	agentSpinner := printer.NewSpinnerWithStatus(
		"Waiting for controlplane agent to become ready...",
		time.Millisecond*500,
		"Controlplane agent is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	agentSpinner.Start()
	defer agentSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods(agentNamespace).
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					LabelSelector:  "app=multicluster-controlplane-agent",
				})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
			conds := make([]metav1.Condition, len(pod.Status.Conditions))
			for i := range pod.Status.Conditions {
				conds[i] = metav1.Condition{
					Type:    string(pod.Status.Conditions[i].Type),
					Status:  metav1.ConditionStatus(pod.Status.Conditions[i].Status),
					Reason:  pod.Status.Conditions[i].Reason,
					Message: pod.Status.Conditions[i].Message,
				}
			}
			return meta.IsStatusConditionTrue(conds, "Ready")
		},
	)
}

// Create bootstrap with token but without CA
func (o *Options) createExternalBootstrapConfig() clientcmdapiv1.Config {
	return clientcmdapiv1.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: "hub",
				Cluster: clientcmdapiv1.Cluster{
					Server:                o.hubAPIServer,
					InsecureSkipTLSVerify: true,
				},
			},
		},
		// Define auth based on the obtained client cert.
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			{
				Name: "bootstrap",
				AuthInfo: clientcmdapiv1.AuthInfo{
					Token: string(o.token),
				},
			},
		},
		// Define a context that connects the auth info and cluster, and set it as the default
		Contexts: []clientcmdapiv1.NamedContext{
			{
				Name: "bootstrap",
				Context: clientcmdapiv1.Context{
					Cluster:   "hub",
					AuthInfo:  "bootstrap",
					Namespace: "default",
				},
			},
		},
		CurrentContext: "bootstrap",
	}
}

func (o *Options) createClientcmdapiv1Config(externalClientUnSecure *kubernetes.Clientset,
	bootstrapExternalConfigUnSecure clientcmdapiv1.Config) (*clientcmdapiv1.Config, error) {
	var err error
	// set hub in cluster endpoint
	if o.forceHubInClusterEndpointLookup {
		o.hubInClusterEndpoint, err = helpers.GetAPIServer(externalClientUnSecure)
		if err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}
		}
	}

	bootstrapConfig := bootstrapExternalConfigUnSecure.DeepCopy()
	bootstrapConfig.Clusters[0].Cluster.InsecureSkipTLSVerify = false
	bootstrapConfig.Clusters[0].Cluster.Server = o.hubAPIServer
	if o.HubCADate != nil {
		// directly set ca-data if --ca-file is set
		bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = o.HubCADate
	} else {
		// get ca data from externalClientUnsecure, ca may empty(cluster-info exists with no ca data)
		ca, err := helpers.GetCACert(externalClientUnSecure)
		if err != nil {
			return nil, err
		}
		bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = ca
	}

	return bootstrapConfig, nil
}

func (o *Options) setKubeconfig() error {
	// replace apiserver if the flag is set, the apiserver value should not be set
	// to in-cluster endpoint until preflight check is finished
	if o.forceHubInClusterEndpointLookup {
		o.HubConfig.Clusters[0].Cluster.Server = o.hubInClusterEndpoint
	}

	bootstrapConfigBytes, err := yaml.Marshal(o.HubConfig)
	if err != nil {
		return err
	}

	o.values.Hub.KubeConfig = base64.StdEncoding.EncodeToString(bootstrapConfigBytes)
	return nil
}
