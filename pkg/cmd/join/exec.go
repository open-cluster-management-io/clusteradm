// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"fmt"
	ocmfeature "open-cluster-management.io/api/feature"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/stolostron/applier/pkg/apply"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/preflight"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
	"open-cluster-management.io/clusteradm/pkg/helpers/version"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

const (
	OperatorNamesapce         = "open-cluster-management"
	DefaultOperatorName       = "klusterlet"
	InstallModeDefault        = "Default"
	InstallModeHosted         = "Hosted"
	KlusterletNamespacePrefix = "open-cluster-management-"
)

func format(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
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
	klog.V(1).InfoS("join options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, "api-server", o.hubAPIServer, "output", o.outputFile)

	// convert mode string to lower
	o.mode = format(o.mode)

	// operatorNamespace is the namespace to deploy klsuterlet;
	// agentNamespace is the namesapce to deploy the agents(registration agent, work agent, etc.);
	// klusterletNamespace is the namespace created on the managed cluster for each klusterlet.
	//
	// The operatorNamespace is fixed to "open-cluster-management".
	// In default mode, agentNamespace is "open-cluster-management-agent", klusterletNamespace refers to agentNamespace, all of these three namesapces are on the managed cluster;
	// In hosted mode, operatorNamespace is on the management cluster, agentNamesapce is "<cluster name>-<6-bit random string>" on the management cluster, and the klusterletNamespace is "open-cluster-management-<agentNamespace>" on the managed cluster.

	// values for default mode
	klusterletName := DefaultOperatorName
	agentNamespace := KlusterletNamespacePrefix + "agent"
	klusterletNamespace := agentNamespace
	if o.mode == InstallModeHosted {
		// add hash suffix to avoid conflict
		klusterletName += "-hosted-" + helpers.RandStringRunes_az09(6)
		agentNamespace = klusterletName
		klusterletNamespace = KlusterletNamespacePrefix + agentNamespace
	}

	o.values = Values{
		ClusterName: o.clusterName,
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
		Registry: o.registry,
		Klusterlet: Klusterlet{
			Mode:                o.mode,
			Name:                klusterletName,
			AgentNamespace:      agentNamespace,
			KlusterletNamespace: klusterletNamespace,
		},
		ManagedKubeconfig:    o.managedKubeconfigFile,
		RegistrationFeatures: genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeRegistrationFeatureGates),
		WorkFeatures:         genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeWorkFeatureGates),
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
	}
	klog.V(3).InfoS("Image version:",
		"'registration image version'", versionBundle.Registration,
		"'placement image version'", versionBundle.Placement,
		"'work image version'", versionBundle.Work,
		"'operator image version'", versionBundle.Operator)

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
		o.values.ManagedKubeconfig = string(managedConfig)
	}

	return nil
}

func (o *Options) run() error {
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	available, err := checkIfRegistrationOperatorAvailable(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := apply.NewApplierBuilder()
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

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
		"join/namespace_agent.yaml",
		"join/bootstrap_hub_kubeconfig.yaml",
	)

	if o.mode == InstallModeHosted {
		files = append(files,
			"join/hosted/external_managed_kubeconfig.yaml",
		)
	}

	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !available {
		operator := "join/operator.yaml"
		out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", operator)
		if err != nil {
			return err
		}
		output = append(output, out...)
	}

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}

	cr := "join/klusterlets.cr.yaml"
	out, err = applier.ApplyCustomResources(reader, o.values, o.ClusteradmFlags.DryRun, "", cr)
	if err != nil {
		return err
	}
	output = append(output, out...)

	klusterletNamespace := o.values.Klusterlet.KlusterletNamespace
	agentNamespace := o.values.Klusterlet.AgentNamespace

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

	fmt.Printf("Please log onto the hub cluster and run the following command:\n\n"+
		"    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), o.values.ClusterName)

	return apply.WriteOutput(o.outputFile, output)

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

	o.values.Hub.KubeConfig = string(bootstrapConfigBytes)
	return nil
}
