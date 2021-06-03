// Copyright Contributors to the Open Cluster Management project
package join

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {

	o.values = Values{
		ClusterName: o.clusterName,
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
	}
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

	return nil
}

func (o *Options) run() error {
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

	kubeClient, err := o.factory.KubernetesClientSet()
	if err != nil {
		return err
	}
	dynamicClient, err := o.factory.DynamicClient()
	if err != nil {
		return err
	}

	restConfig, err := o.factory.ToRESTConfig()
	if err != nil {
		return err
	}

	clientHolder := resourceapply.NewClientHolder().
		WithAPIExtensionsClient(apiextensionsclient.NewForConfigOrDie(restConfig)).
		WithKubernetes(kubeClient).
		WithDynamicClient(dynamicClient)

	files := []string{
		"join/namespace_agent.yaml",
		"join/namespace.yaml",
		"join/bootstrap_hub_kubeconfig.yaml",
		"join/cluster_role.yaml",
		"join/cluster_role_binding.yaml",
		"join/klusterlets.crd.yaml",
		"join/service_account.yaml",
	}

	err = helpers.ApplyDirectly(clientHolder, reader, scenarioDirectory, o.values, files...)
	if err != nil {
		return err
	}

	err = helpers.ApplyDeployment(kubeClient, reader, scenarioDirectory, o.values, "join/operator.yaml")
	if err != nil {
		return err
	}

	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	err = helpers.ApplyCustomResouces(dynamicClient, discoveryClient, reader, scenarioDirectory, o.values, "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}
	fmt.Printf("login back onto the hub and run: %s accept --clusters %s\n", helpers.GetExampleHeader(), o.values.ClusterName)

	return nil

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
		// fmt.Printf("Failed to get CA")
		return "", err
	}

	hubAPIServerInternal, err := helpers.GetAPIServer(externalClientUnSecure)
	if err != nil {
		if errors.IsNotFound(err) {
			hubAPIServerInternal = o.hubAPIServer
		} else {
			// fmt.Printf("Failed to GetAPIServer")
			return "", err
		}
	}

	bootstrapConfig := bootstrapExternalConfigUnSecure
	bootstrapConfig.Clusters[0].Cluster.InsecureSkipTLSVerify = false
	bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = ca
	bootstrapConfig.Clusters[0].Cluster.Server = hubAPIServerInternal
	bootstrapConfigBytes, err := yaml.Marshal(bootstrapConfig)
	if err != nil {
		return "", err
	}

	return string(bootstrapConfigBytes), nil
}
