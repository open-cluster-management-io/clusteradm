// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"
	"time"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
		},
	}
	return nil
}

func (o *Options) validate() error {
	return nil
}

func (o *Options) run() error {
	reader := scenario.GetScenarioResourcesReader()

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
		"init/bootstrap-token-secret.yaml",
		"init/cluster_role_bootstrap.yaml",
		"init/cluster_role_binding_bootstrap.yaml",
		"init/cluster_role.yaml",
		"init/cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/namespace.yaml",
		"init/service_account.yaml",
	}

	err = apply.ApplyDirectly(clientHolder, reader, o.values, "", files...)
	if err != nil {
		return err
	}

	err = apply.ApplyDeployment(kubeClient, reader, o.values, "", "init/operator.yaml")
	if err != nil {
		return err
	}
	//quick fix for https://github.com/open-cluster-management-io/clusteradm/issues/12
	fmt.Printf("Wait 10 sec... for the crd to be effective\n")
	time.Sleep(10 * time.Second)

	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	err = apply.ApplyCustomResouces(dynamicClient, discoveryClient, reader, o.values, "", "init/clustermanagers.cr.yaml")
	if err != nil {
		return err
	}

	fmt.Printf("login into the cluster and run: %s join --hub-token %s.%s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		o.values.Hub.TokenID,
		o.values.Hub.TokenSecret,
		restConfig.Host,
	)

	return nil
}
