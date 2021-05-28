// Copyright Contributors to the Open Cluster Management project
package hub

import (
	"fmt"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/hub/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.values = make(map[string]interface{})
	hub := make(map[string]interface{})
	hub["tokenID"] = helpers.RandStringRunes_az09(6)
	hub["tokenSecret"] = helpers.RandStringRunes_az09(16)
	o.values["hub"] = hub
	return nil
}

func (o *Options) validate() error {
	return nil
}

func (o *Options) run() error {
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) error {
	// ss := &corev1.SecretList{}
	// ls := labels.SelectorFromSet(labels.Set{
	// 	"app": "cluster-manager",
	// })
	// err := client.List(context.TODO(),
	// 	ss,
	// 	&crclient.ListOptions{
	// 		LabelSelector: ls,
	// 		Namespace:     "kube-system",
	// 	})
	// if err != nil {
	// 	return err
	// }
	// var bootstrapSecret *corev1.Secret
	// for _, item := range ss.Items {
	// 	if strings.HasPrefix(item.Name, "bootstrap-token") {
	// 		bootstrapSecret = &item
	// 		break
	// 	}
	// }
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

	files, err := reader.AssetNames([]string{"init/hub/operator.yaml"})
	if err != nil {
		return err
	}
	fmt.Printf("file: %v", files)
	// if bootstrapSecret == nil {
	// 	err = applyOptions.ApplyWithValues(client, reader,
	// 		filepath.Join(scenarioDirectory, "hub"), []string{},
	// 		o.values)
	// } else {
	// 	o.values["hub"].(map[string]interface{})["tokenID"] = string(bootstrapSecret.Data["token-id"])
	// 	o.values["hub"].(map[string]interface{})["tokenSecret"] = string(bootstrapSecret.Data["token-secret"])
	// 	err = applyOptions.ApplyWithValues(client, reader,
	// 		filepath.Join(scenarioDirectory, "hub"), []string{"boostrap-token-secret.yaml"},
	// 		o.values)
	// }

	resourceResults := helpers.ApplyDirectly(clientHolder, reader, scenarioDirectory, o.values, files...)

	errs := []error{}
	for _, result := range resourceResults {
		if result.Error != nil {
			errs = append(errs, fmt.Errorf("%q (%T): %v", result.File, result.Type, result.Error))
		}
	}

	fmt.Printf("errors: %v\n", errs)

	_, err = helpers.ApplyDeployment(kubeClient, nil, reader, scenarioDirectory, o.values, "init/hub/operator.yaml")
	if err != nil {
		return err
	}
	apiServerInternal, err := helpers.GetAPIServer(client)
	if err != nil {
		return err
	}

	fmt.Printf("login into the cluster and run: %s join hub --hub-token %s.%s --hub-server-internal %s --hub-server-external %s --name <cluster_name>\n",
		helpers.GetExampleHeader(),
		o.values["hub"].(map[string]interface{})["tokenID"].(string),
		o.values["hub"].(map[string]interface{})["tokenSecret"].(string),
		apiServerInternal,
		restConfig.Host,
	)

	return nil
}
