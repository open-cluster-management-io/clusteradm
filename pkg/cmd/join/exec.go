// Copyright Contributors to the Open Cluster Management project
package join

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"reflect"
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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/preflight"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
	"open-cluster-management.io/clusteradm/pkg/helpers/resourcerequirement"
	"open-cluster-management.io/clusteradm/pkg/helpers/wait"
	klusterletchart "open-cluster-management.io/ocm/deploy/klusterlet/chart"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
	sdkhelpers "open-cluster-management.io/sdk-go/pkg/helpers"
)

const (
	AgentNamespacePrefix = "open-cluster-management-"

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

	o.klusterletChartConfig.Klusterlet = klusterletchart.KlusterletConfig{
		ClusterName: o.clusterName,
	}

	o.klusterletChartConfig.Images = klusterletchart.ImagesConfig{
		Registry: o.registry,
		ImageCredentials: klusterletchart.ImageCredentials{
			CreateImageCredentials: true,
		},
	}
	o.klusterletChartConfig.EnableSyncLabels = o.enableSyncLabels

	if o.imagePullCredFile != "" {
		_, err := os.ReadFile(o.imagePullCredFile)
		if err != nil {
			return fmt.Errorf("failed read the image pull credential file %v: %v", o.imagePullCredFile, err)
		}
		// TODO add user/password
	}

	// deploy klusterlet
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
	if o.mode == string(operatorv1.InstallModeHosted) {
		// add hash suffix to avoid conflict
		klusterletName += "-hosted-" + helpers.RandStringRunes_az09(6)
		agentNamespace = klusterletName
		klusterletNamespace = AgentNamespacePrefix + agentNamespace
	}

	o.klusterletChartConfig.Klusterlet.Name = klusterletName
	o.klusterletChartConfig.Klusterlet.Namespace = klusterletNamespace

	resourceRequirement, err := resourcerequirement.NewResourceRequirement(
		operatorv1.ResourceQosClass(o.resourceQosClass), o.resourceLimits, o.resourceRequests)
	if err != nil {
		return err
	}
	if resourceRequirement.ResourceRequirements != nil {
		o.klusterletChartConfig.Resources = *resourceRequirement.ResourceRequirements
	}

	o.klusterletChartConfig.Klusterlet.RegistrationConfiguration = operatorv1.RegistrationConfiguration{
		FeatureGates: genericclioptionsclusteradm.ConvertToFeatureGateAPI(
			genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeRegistrationFeatureGates),
		ClientCertExpirationSeconds: o.clientCertExpirationSeconds,
	}
	o.klusterletChartConfig.Klusterlet.WorkConfiguration = operatorv1.WorkAgentConfiguration{
		FeatureGates: genericclioptionsclusteradm.ConvertToFeatureGateAPI(
			genericclioptionsclusteradm.SpokeMutableFeatureGate, ocmfeature.DefaultSpokeWorkFeatureGates),
	}

	// set mode based on mode and singleton
	if o.mode == string(operatorv1.InstallModeHosted) && o.singleton {
		o.klusterletChartConfig.Klusterlet.Mode = operatorv1.InstallModeSingletonHosted
	} else if o.singleton {
		o.klusterletChartConfig.Klusterlet.Mode = operatorv1.InstallModeSingleton
	} else {
		o.klusterletChartConfig.Klusterlet.Mode = operatorv1.InstallMode(o.mode)
	}

	o.klusterletChartConfig.Images.Tag = o.bundleVersion

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

	// Create an unsecure bootstrap
	bootstrapExternalConfigUnSecure := o.createExternalBootstrapConfig()
	// create external client from the bootstrap
	externalClientUnSecure, err := helpers.CreateClientFromClientcmdapiv1Config(bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}
	// Create the kubeconfig for the internal client
	o.HubConfig, err = o.createClientcmdapiv1Config(externalClientUnSecure, bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}

	// get managed cluster externalServerURL
	var kubeClient *kubernetes.Clientset
	switch o.mode {
	case string(operatorv1.InstallModeHosted):
		restConfig, err := clientcmd.BuildConfigFromFlags("", o.managedKubeconfigFile)
		if err != nil {
			return err
		}
		kubeClient, err = kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}
	default:
		kubeClient, err = o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
		if err != nil {
			klog.Errorf("Failed building kube client: %v", err)
			return err
		}
	}

	klusterletApiserver, err := sdkhelpers.GetAPIServer(kubeClient)
	if err != nil {
		klog.Warningf("Failed looking for cluster endpoint for the registering klusterlet: %v", err)
		klusterletApiserver = ""
	} else if !preflight.ValidAPIHost(klusterletApiserver) {
		klog.Warningf("ConfigMap/cluster-info.data.kubeconfig.clusters[0].cluster.server field [%s] in namespace kube-public should start with http:// or https://", klusterletApiserver)
		klusterletApiserver = ""
	}
	o.klusterletChartConfig.Klusterlet.ExternalServerURLs = []operatorv1.ServerURL{
		{
			URL: klusterletApiserver,
		},
	}

	if err := o.capiOptions.Complete(cmd, args); err != nil {
		return err
	}
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
				ClusterName: o.klusterletChartConfig.Klusterlet.ClusterName,
			},
		}, os.Stderr); err != nil {
		return err
	}

	err := o.setKubeconfig()
	if err != nil {
		return err
	}

	// get ManagedKubeconfig from given file
	if o.mode == string(operatorv1.InstallModeHosted) {
		managedConfig, err := os.ReadFile(o.managedKubeconfigFile)
		if err != nil {
			return err
		}

		// replace the server address with the internal endpoint
		if o.forceManagedInClusterEndpointLookup {
			config := &clientcmdapiv1.Config{}
			err = yaml.Unmarshal(managedConfig, config)
			if err != nil {
				return err
			}
			restConfig, err := clientcmd.BuildConfigFromFlags("", o.managedKubeconfigFile)
			if err != nil {
				return err
			}
			kubeClient, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return err
			}
			inClusterEndpoint, err := sdkhelpers.GetAPIServer(kubeClient)
			if err != nil {
				return err
			}
			config.Clusters[0].Cluster.Server = inClusterEndpoint
			managedConfig, err = yaml.Marshal(config)
			if err != nil {
				return err
			}
		}
		o.klusterletChartConfig.ExternalManagedKubeConfig = base64.StdEncoding.EncodeToString(managedConfig)
	}

	if err := o.capiOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *Options) run() error {
	f := o.ClusteradmFlags.KubectlFactory
	if o.capiOptions.Enable {
		getter, err := o.capiOptions.ToClientGetter()
		if err != nil {
			return err
		}
		f = util.NewFactory(getter)
	}

	_, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}

	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	operatorClient, err := operatorclient.NewForConfig(config)
	if err != nil {
		return err
	}

	r := reader.NewResourceReader(f, o.ClusteradmFlags.DryRun, o.Streams)

	if err = o.applyKlusterlet(r, operatorClient, apiExtensionsClient); err != nil {
		return err
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
		"    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), o.klusterletChartConfig.Klusterlet.ClusterName)
	fmt.Fprintf(o.Streams.Out, "This is not needed when the ManagedClusterAutoApproval feature is enabled\n")
	return nil

}

func (o *Options) applyKlusterlet(r *reader.ResourceReader, operatorClient operatorclient.Interface, apiExtensionsClient apiextensionsclient.Interface) error {
	available, err := checkIfRegistrationOperatorAvailable(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	// If Deployment/klusterlet is not deployed, deploy it
	if !available {
		o.klusterletChartConfig.CreateNamespace = o.createNameSpace

		raw, err := chart.RenderKlusterletChart(
			o.klusterletChartConfig,
			"open-cluster-management")
		if err != nil {
			return err
		}

		if err := r.ApplyRaw(raw); err != nil {
			return err
		}
	}

	if !o.ClusteradmFlags.DryRun {
		if err := wait.WaitUntilCRDReady(apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}

	if !available && o.wait && !o.ClusteradmFlags.DryRun {
		err = waitUntilRegistrationOperatorConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout))
		if err != nil {
			return err
		}
	}

	if o.wait && !o.ClusteradmFlags.DryRun {
		if o.mode == string(operatorv1.InstallModeHosted) {
			err = waitUntilKlusterletConditionIsTrue(operatorClient, int64(o.ClusteradmFlags.Timeout), o.klusterletChartConfig.Klusterlet.Name)
			if err != nil {
				return err
			}
		} else {
			err = waitUntilKlusterletConditionIsTrue(operatorClient, int64(o.ClusteradmFlags.Timeout), o.klusterletChartConfig.Klusterlet.Name)
			if err != nil {
				return err
			}
		}

		err = o.waitUntilManagedClusterIsCreated(int64(o.ClusteradmFlags.Timeout), o.klusterletChartConfig.Klusterlet.ClusterName)
		if err != nil {
			return err
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

func (o *Options) waitUntilManagedClusterIsCreated(timeout int64, clusterName string) error {
	// Create an unsecure bootstrap
	bootstrapExternalConfigUnSecure := o.createExternalBootstrapConfig()
	restConfig, err := helpers.CreateRESTConfigFromClientcmdapiv1Config(bootstrapExternalConfigUnSecure)
	if err != nil {
		return fmt.Errorf("failed to create rest config: %v", err)
	}
	clusterClient, err := clusterclient.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	phase := &atomic.Value{}
	phase.Store("")
	operatorSpinner := printer.NewSpinnerWithStatus(
		"Waiting for managed cluster to be created...",
		time.Millisecond*500,
		"Managed cluster is created.\n",
		func() string {
			return phase.Load().(string)
		})
	operatorSpinner.Start()
	defer operatorSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			w, err := clusterClient.ClusterV1().ManagedClusters().
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					FieldSelector:  fmt.Sprintf("metadata.name=%s", clusterName),
				})
			if err != nil {
				return nil, fmt.Errorf("failed to watch: %v", err)
			}
			return w, nil
		},
		func(event watch.Event) bool {
			cluster, ok := event.Object.(*clusterv1.ManagedCluster)
			if !ok {
				return false
			}
			return cluster.Name == clusterName
		})
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
func waitUntilKlusterletConditionIsTrue(client operatorclient.Interface, timeout int64, klusterletName string) error {
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
			return client.OperatorV1().Klusterlets().
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					FieldSelector:  fmt.Sprintf("metadata.name=%s", klusterletName),
				})
		},
		func(event watch.Event) bool {
			klusterlet, ok := event.Object.(*operatorv1.Klusterlet)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerKlusterletStatus(klusterlet))
			return meta.IsStatusConditionFalse(klusterlet.Status.Conditions, "RegistrationDesiredDegraded") &&
				meta.IsStatusConditionFalse(klusterlet.Status.Conditions, "WorkDesiredDegraded")
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
					Token: o.token,
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
		o.hubInClusterEndpoint, err = sdkhelpers.GetAPIServer(externalClientUnSecure)
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
		ca, err := sdkhelpers.GetCACert(externalClientUnSecure)
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

	// set the proxy url
	if len(o.proxyURL) > 0 {
		o.HubConfig.Clusters[0].Cluster.ProxyURL = o.proxyURL
	}

	// append the proxy ca data
	if len(o.proxyURL) > 0 && len(o.proxyCAFile) > 0 {
		proxyCAData, err := os.ReadFile(o.proxyCAFile)
		if err != nil {
			return err
		}
		o.HubConfig.Clusters[0].Cluster.CertificateAuthorityData, err = mergeCertificateData(
			o.HubConfig.Clusters[0].Cluster.CertificateAuthorityData, proxyCAData)
		if err != nil {
			return err
		}
	}

	bootstrapConfigBytes, err := yaml.Marshal(o.HubConfig)
	if err != nil {
		return err
	}

	o.klusterletChartConfig.BootstrapHubKubeConfig = string(bootstrapConfigBytes)
	return nil
}

func mergeCertificateData(caBundles ...[]byte) ([]byte, error) {
	var all []*x509.Certificate
	for _, caBundle := range caBundles {
		if len(caBundle) == 0 {
			continue
		}
		certs, err := certutil.ParseCertsPEM(caBundle)
		if err != nil {
			return []byte{}, err
		}
		all = append(all, certs...)
	}

	// remove duplicated cert
	var merged []*x509.Certificate
	for i := range all {
		found := false
		for j := range merged {
			if reflect.DeepEqual(all[i].Raw, merged[j].Raw) {
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, all[i])
		}
	}

	// encode the merged certificates
	b := bytes.Buffer{}
	for _, cert := range merged {
		if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return []byte{}, err
		}
	}
	return b.Bytes(), nil
}
