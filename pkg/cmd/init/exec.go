// Copyright Contributors to the Open Cluster Management project
package init

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
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
	token := fmt.Sprintf("%s.%s", o.values.Hub.TokenID, o.values.Hub.TokenSecret)
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
		"init/namespace.yaml",
	}
	if o.useBootstrapToken {
		files = append(files,
			"init/bootstrap-token-secret.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_cluster_role_binding.yaml",
		)
	} else {
		files = append(files,
			"init/bootstrap_sa.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_sa_cluster_role_binding.yaml",
		)
	}

	files = append(files,
		"init/clustermanager_cluster_role.yaml",
		"init/clustermanager_cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/clustermanager_sa.yaml",
	)

	out, err := apply.ApplyDirectly(clientHolder, reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.useBootstrapToken {
		b := retry.DefaultBackoff
		b.Duration = 100 * time.Millisecond
		secret, err := waitForBootstrapSecret(kubeClient, b)
		if err != nil {
			return err
		}
		token = string(secret.Data["token"])
	}
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
	out, err = apply.ApplyCustomResouces(dynamicClient, discoveryClient, reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanager.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	fmt.Printf("please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		token,
		restConfig.Host,
	)

	return apply.WriteOutput(o.outputFile, output)
}

func waitForBootstrapSecret(kubeClient kubernetes.Interface, b wait.Backoff) (secret *corev1.Secret, err error) {
	err = retry.OnError(b, func(err error) bool {
		if err != nil {
			fmt.Printf("Wait for sa %s secret to be ready\n", config.BootstrapSAName)
			return true
		}
		return false
	}, func() error {
		secret, err = helpers.GetBootstrapSecret(kubeClient)
		return err
	})
	return
}
