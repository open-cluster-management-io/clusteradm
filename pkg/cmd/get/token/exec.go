// Copyright Contributors to the Open Cluster Management project
package token

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stolostron/applier/pkg/apply"
	"github.com/stolostron/applier/pkg/asset"
	"k8s.io/apimachinery/pkg/api/errors"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
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

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
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
		token, err = helpers.GetBootstrapToken(context.TODO(), kubeClient)
	} else {
		token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
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
		token, err = helpers.GetBootstrapToken(context.TODO(), kubeClient)
		if err != nil {
			return err
		}
		return o.writeResult(token, restConfig.Host, output)
	}

	//read the token
	token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
	if err != nil {
		return err
	}

	return o.writeResult(token, restConfig.Host, output)
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
