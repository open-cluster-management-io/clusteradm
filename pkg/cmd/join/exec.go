// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"

	// "k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/retry"
	operatorclientv1 "open-cluster-management.io/api/client/operator/clientset/versioned/typed/operator/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/cmd/join/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("join options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, "api-server", o.hubAPIServer, o.outputFile)

	o.values = Values{
		ClusterName: o.clusterName,
		Hub: Hub{
			APIServer: o.hubAPIServer,
		},
	}
	klog.V(3).InfoS("values:", "clusterName", o.values.ClusterName, "hubAPIServer", o.values.Hub.APIServer)
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
		b := retry.DefaultBackoff
		b.Duration = 200 * time.Millisecond

		err = helpers.WaitCRDToBeReady(
			apiExtensionsClient, "klusterlets.operator.open-cluster-management.io", b)
		if err != nil {
			return err
		}
	}

	out, err = applier.ApplyCustomResources(reader, o.values, o.ClusteradmFlags.DryRun, "", "join/klusterlets.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	err = waitUntilConditionIsTrue(o.ClusteradmFlags.KubectlFactory, 300)
	if err != nil {
		return err
	}

	fmt.Printf("Please log onto the hub cluster and run the following command:\n\n"+
		"    %s accept --clusters %s\n\n", helpers.GetExampleHeader(), o.values.ClusterName)

	return apply.WriteOutput(o.outputFile, output)

}

//Wait until the klusterlet condition available=true, or timeout in $timeout seconds
func waitUntilConditionIsTrue(f util.Factory, timeout int64) error {
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

	watch, err := client.Klusterlets().Watch(context.TODO(), metav1.ListOptions{TimeoutSeconds: &timeout, FieldSelector: "metadata.name=klusterlet"})
	if err != nil {
		return err
	}
	defer watch.Stop()

	fmt.Printf("Waiting for the management components to become ready...\n")

	for {
		event, ok := <-watch.ResultChan()
		if !ok { //The channel is closed by Kubernetes, thus, user should check the pod status manually
			return fmt.Errorf("the registration agent is not ready yet, please check it manually")
		}

		klusterlet, ok := event.Object.(*operatorv1.Klusterlet)
		if !ok {
			continue
		}

		if meta.IsStatusConditionTrue(klusterlet.Status.Conditions, "Available") {
			break
		}
	}
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
