// Copyright Contributors to the Open Cluster Management project
package join

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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
			ExternalServerURL: o.hubServerExternal,
			InternalServerURL: o.hubServerInternal,
		},
	}
	if o.values.Hub.InternalServerURL == "" {
		o.values.Hub.InternalServerURL = o.values.Hub.ExternalServerURL
	}
	return nil
}

func (o *Options) validate() error {
	if o.token == "" {
		return fmt.Errorf("token is missing")
	}
	if o.values.Hub.ExternalServerURL == "" {
		return fmt.Errorf("hub-server is misisng")
	}
	if o.values.ClusterName == "" {
		return fmt.Errorf("name is missing")
	}

	return nil
}

func (o *Options) run() error {
	reader := scenario.GetScenarioResourcesReader()

	bootstrapConfigUnSecure := clientcmdapiv1.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: "hub",
				Cluster: clientcmdapiv1.Cluster{
					Server:                o.hubServerExternal,
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

	bootstrapConfigBytesUnSecure, err := yaml.Marshal(bootstrapConfigUnSecure)
	if err != nil {
		return err
	}

	configUnSecure, err := clientcmd.Load(bootstrapConfigBytesUnSecure)
	if err != nil {
		return err
	}
	restConfigUnSecure, err := clientcmd.NewDefaultClientConfig(*configUnSecure, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return err
	}

	clientUnSecure, err := kubernetes.NewForConfig(restConfigUnSecure)
	if err != nil {
		return err
	}

	ca, err := helpers.GetCACert(clientUnSecure)
	if err != nil {
		return err
	}

	bootstrapConfig := bootstrapConfigUnSecure
	bootstrapConfig.Clusters[0].Cluster.InsecureSkipTLSVerify = false
	bootstrapConfig.Clusters[0].Cluster.CertificateAuthorityData = ca
	bootstrapConfig.Clusters[0].Cluster.Server = o.values.Hub.InternalServerURL
	bootstrapConfigBytes, err := yaml.Marshal(bootstrapConfig)
	if err != nil {
		return err
	}

	o.values.Hub.KubeConfig = string(bootstrapConfigBytes)

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
	err = helpers.ApplyCustomResouces(dynamicClient, reader, scenarioDirectory, o.values, "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}
	fmt.Printf("login back onto the hub and run: %s accept clusters --names %s\n", helpers.GetExampleHeader(), o.values.ClusterName)

	return nil

}
