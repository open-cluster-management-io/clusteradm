// Copyright Contributors to the Open Cluster Management project
package token

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	clusteradmjson "open-cluster-management.io/clusteradm/pkg/helpers/json"
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
	kubeClient, _, _, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	var token string

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
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
