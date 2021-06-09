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
	"k8s.io/client-go/util/retry"
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
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}
	dynamicClient, err := o.ClusteradmFlags.KubectlFactory.DynamicClient()
	if err != nil {
		return err
	}

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	clientHolder := resourceapply.NewClientHolder().
		WithAPIExtensionsClient(apiExtensionsClient).
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

	out, err := apply.ApplyDirectly(clientHolder, reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	out, err = apply.ApplyDeployments(kubeClient, reader, o.values, o.ClusteradmFlags.DryRun, "", "init/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.ClusteradmFlags.DryRun {
		b := retry.DefaultBackoff
		b.Duration = 100 * time.Millisecond
		err = helpers.WaitCRDToBeReady(*apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", b)
		if err != nil {
			return err
		}
	}

	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	out, err = apply.ApplyCustomResouces(dynamicClient, discoveryClient, reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanagers.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	fmt.Printf("login into the cluster and run: %s join --hub-token %s.%s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		o.values.Hub.TokenID,
		o.values.Hub.TokenSecret,
		restConfig.Host,
	)

	return apply.WriteOutput(o.outputFile, output)
}
