// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"

	operatorclientv1 "open-cluster-management.io/api/client/operator/clientset/versioned/typed/operator/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("join options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, "api-server", o.hubAPIServer, o.outputFile)

	o.values = Values{
		ClusterName: o.clusterName,
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
		ImageRegistry: ImageRegistry{
			Registry: o.registry,
			Version:  o.version,
		},
	}
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		klog.Errorf("Failed building kube client: %v", err)
		return err
	}
	klusterletApiserver, err := helpers.GetAPIServer(kubeClient)
	if err != nil {
		klog.Errorf("Failed looking for cluster endpoint for the registering klusterlet: %v", err)
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
	if o.token == "" {
		return fmt.Errorf("token is missing")
	}
	if o.values.Hub.APIServer == "" {
		return fmt.Errorf("hub-server is misisng")
	}
	if o.values.ClusterName == "" {
		return fmt.Errorf("name is missing")
	}
	if len(o.registry) == 0 {
		return fmt.Errorf("registry should not be empty")
	}
	if len(o.version) == 0 {
		return fmt.Errorf("version should not be empty")
	}

	return nil
}

func (o *Options) run() error {
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	//Create an unsecure bootstrap
	bootstrapExternalConfigUnSecure := o.createExternalBootstrapConfig()

	//create external client from the bootstrap
	externalClientUnSecure, err := createExternalClientFromBootstrap(bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}

	//Create the kubeconfig for the internal client
	o.values.Hub.KubeConfig, err = o.createKubeConfig(externalClientUnSecure, bootstrapExternalConfigUnSecure)
	if err != nil {
		return err
	}

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}
	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"join/namespace_agent.yaml",
		"join/namespace_addon.yaml",
		"join/namespace.yaml",
		"join/bootstrap_hub_kubeconfig.yaml",
		"join/cluster_role.yaml",
		"join/cluster_role_binding.yaml",
		"join/klusterlets.crd.yaml",
		"join/service_account.yaml",
	}

	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", "join/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.ClusteradmFlags.DryRun {
		if o.wait && !o.ClusteradmFlags.DryRun {
			if err := waitUntilCRDReady(apiExtensionsClient); err != nil {
				return err
			}
		}
	}

	out, err = applier.ApplyCustomResources(reader, o.values, o.ClusteradmFlags.DryRun, "", "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if o.wait && !o.ClusteradmFlags.DryRun {
		err = waitUntilRegistrationOperatorConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout))
		if err != nil {
			return err
		}
	}

	if o.wait && !o.ClusteradmFlags.DryRun {
		err = waitUntilKlusterletConditionIsTrue(o.ClusteradmFlags.KubectlFactory, int64(o.ClusteradmFlags.Timeout))
		if err != nil {
			return err
		}
	}

	fmt.Printf("Please log onto the hub cluster and run the following command:\n\n"+
		"    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), o.values.ClusterName)

	return apply.WriteOutput(o.outputFile, output)

}

func waitUntilCRDReady(apiExtensionsClient clientset.Interface) error {
	b := retry.DefaultBackoff
	b.Duration = 200 * time.Millisecond

	crdSpinner := helpers.NewSpinner("Waiting for CRD to be ready...", time.Second)
	crdSpinner.FinalMSG = "CRD successfully registered.\n"
	crdSpinner.Start()
	defer crdSpinner.Stop()
	return helpers.WaitCRDToBeReady(
		apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", b)
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
	operatorSpinner := helpers.NewSpinner("Waiting for registration operator to become ready...", time.Millisecond*500)
	operatorSpinner.FinalMSG = "Registration operator is now available.\n"
	operatorSpinner.Start()
	defer operatorSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods("open-cluster-management").
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

//Wait until the klusterlet condition available=true, or timeout in $timeout seconds
func waitUntilKlusterletConditionIsTrue(f util.Factory, timeout int64) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}

	var client *operatorclientv1.OperatorV1Client
	client, err = operatorclientv1.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	klusterletSpinner := helpers.NewSpinner("Waiting for klusterlet agent to become ready...", time.Millisecond*500)
	klusterletSpinner.FinalMSG = "Klusterlet is now available.\n"
	klusterletSpinner.Start()
	defer klusterletSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.Klusterlets().
				Watch(
					context.TODO(),
					metav1.ListOptions{
						TimeoutSeconds: &timeout,
						FieldSelector:  "metadata.name=klusterlet",
					})
		},
		func(event watch.Event) bool {
			klusterlet, ok := event.Object.(*operatorv1.Klusterlet)
			if !ok {
				return false
			}
			return meta.IsStatusConditionTrue(klusterlet.Status.Conditions, "Available")
		},
	)
}

//Create bootstrap with token but without CA
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

func createExternalClientFromBootstrap(bootstrapExternalConfigUnSecure clientcmdapiv1.Config) (*kubernetes.Clientset, error) {
	bootstrapConfigBytesUnSecure, err := yaml.Marshal(bootstrapExternalConfigUnSecure)
	if err != nil {
		return nil, err
	}

	configUnSecure, err := clientcmd.Load(bootstrapConfigBytesUnSecure)
	if err != nil {
		return nil, err
	}
	restConfigUnSecure, err := clientcmd.NewDefaultClientConfig(*configUnSecure, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}

	clientUnSecure, err := kubernetes.NewForConfig(restConfigUnSecure)
	if err != nil {
		return nil, err
	}
	return clientUnSecure, nil
}

func (o *Options) createKubeConfig(externalClientUnSecure *kubernetes.Clientset,
	bootstrapExternalConfigUnSecure clientcmdapiv1.Config) (string, error) {
	ca, err := helpers.GetCACert(externalClientUnSecure)
	if err != nil {
		return "", err
	}

	hubAPIServerInternal, err := helpers.GetAPIServer(externalClientUnSecure)
	if err != nil {
		if errors.IsNotFound(err) {
			hubAPIServerInternal = o.hubAPIServer
		} else {
			return "", err
		}
	}

	bootstrapConfig := bootstrapExternalConfigUnSecure.DeepCopy()
	bootstrapConfig.Clusters[0].Cluster.InsecureSkipTLSVerify = false
	bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = ca
	if !o.skipHubInClusterEndpointLookup {
		bootstrapConfig.Clusters[0].Cluster.Server = hubAPIServerInternal
	}
	bootstrapConfigBytes, err := yaml.Marshal(bootstrapConfig)
	if err != nil {
		return "", err
	}

	return string(bootstrapConfigBytes), nil
}
