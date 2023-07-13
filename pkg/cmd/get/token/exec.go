// Copyright Contributors to the Open Cluster Management project
package token

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	clusteradmjson "open-cluster-management.io/clusteradm/pkg/helpers/json"
	"open-cluster-management.io/clusteradm/pkg/helpers/reader"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
		},
	}
	f := o.ClusteradmFlags.KubectlFactory
	o.builder = f.NewBuilder()
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
	kubeClient, _, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	r := reader.NewResourceReader(o.builder, o.ClusteradmFlags.DryRun, o.Streams)

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
		err := o.applyToken(r)
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

	err = r.Apply(scenario.Files, o.values, files...)
	if err != nil {
		return err
	}

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	// if dry-run then there is nothing else to do
	if o.ClusteradmFlags.DryRun {
		return o.writeResult(token, restConfig.Host)
	}

	//if bootstrap token then read the token
	if o.useBootstrapToken {
		token, err = helpers.GetBootstrapToken(context.TODO(), kubeClient)
		if err != nil {
			return err
		}
		return o.writeResult(token, restConfig.Host)
	}

	//read the token
	token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
	if err != nil {
		return err
	}

	return o.writeResult(token, restConfig.Host)
}

func (o *Options) applyToken(applier *reader.ResourceReader) error {
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
	return applier.Apply(scenario.Files, o.values, files...)
}

func (o *Options) writeResult(token, host string) error {
	if len(token) == 0 {
		fmt.Println(o.Streams.Out, "token doesn't exist")
		return nil
	}
	if o.output == "json" {
		err := clusteradmjson.WriteJsonOutput(o.Streams.Out, clusteradmjson.HubInfo{
			HubToken:     token,
			HubApiserver: host,
		})
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintf(o.Streams.Out, "token=%s\n", token)
		fmt.Fprintf(o.Streams.Out, "please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n", helpers.GetExampleHeader(), token, host)
	}
	return nil
}
