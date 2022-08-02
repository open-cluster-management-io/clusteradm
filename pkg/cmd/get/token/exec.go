// Copyright Contributors to the Open Cluster Management project
package token

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/stolostron/applier/pkg/apply"
	"github.com/stolostron/applier/pkg/asset"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
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
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsClusterManagerInstalled(apiExtensionsClient)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("this is not a hub")
	}
	return err
}

func (o *Options) run() error {
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := apply.NewApplierBuilder()
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	//Retrieve token from service-account/bootstrap-token
	// and if not found create it
	var token string
	if o.useBootstrapToken {
		token, err = helpers.GetBootstrapToken(kubeClient)
	} else {
		token, err = helpers.GetBootstrapTokenFromSA(kubeClient)
	}
	switch {
	case errors.IsNotFound(err):
		out, err := o.applyToken(applier, reader)
		output = append(output, out...)
		if err != nil {
			return err
		}
	case err != nil:
		return err
	}

	//Update the cluster role as it could change over-time
	files := []string{
		"init/bootstrap_cluster_role.yaml",
	}
	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	// if dry-run then there is nothing else to do
	if o.ClusteradmFlags.DryRun {
		return o.writeResult(token, restConfig.Host, output)
	}

	//if bootstrap token then read the token
	if o.useBootstrapToken {
		token, err = helpers.GetBootstrapToken(kubeClient)
		if err != nil {
			return err
		}
		return o.writeResult(token, restConfig.Host, output)
	}

	//if service-account wait for the sa secret
	err = wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		return waitForBootstrapToken(kubeClient)
	})
	if err != nil {
		return err
	}

	//read the token
	token, err = helpers.GetBootstrapTokenFromSA(kubeClient)
	if err != nil {
		return err
	}

	return o.writeResult(token, restConfig.Host, output)
}

func waitForBootstrapToken(kubeClient kubernetes.Interface) (bool, error) {
	_, err := helpers.GetBootstrapTokenFromSA(kubeClient)
	switch {
	case errors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

func (o *Options) applyToken(applier apply.Applier, reader *asset.ScenarioResourcesReader) ([]string, error) {
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
	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return nil, err
	}
	return out, err
}

func (o *Options) writeResult(token, host string, output []string) error {
	if len(token) == 0 {
		fmt.Println("token doesn't exist")
		return apply.WriteOutput(o.outputFile, output)
	}
	fmt.Printf("token=%s\n", token)
	fmt.Printf("please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n", helpers.GetExampleHeader(), token, host)
	return apply.WriteOutput(o.outputFile, output)
}
