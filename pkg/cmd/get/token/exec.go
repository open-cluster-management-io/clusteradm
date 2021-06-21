// Copyright Contributors to the Open Cluster Management project
package token

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
	"open-cluster-management.io/clusteradm/pkg/helpers/asset"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
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

	//Retrieve token from service-account and if not found
	//from the bootstrap token
	token, err := helpers.GetToken(kubeClient)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		out, err := o.applyToken(clientHolder, reader)
		if err != nil {
			return err
		}
		output = append(output, out...)
		if !o.ClusteradmFlags.DryRun {
			//Make sure that the sa token is ready
			if !o.useBootstrapToken {
				b := retry.DefaultBackoff
				b.Duration = 100 * time.Millisecond
				_, err = waitForBootstrapSecret(kubeClient, b)
				if err != nil {
					return err
				}
			}
			token, err = helpers.GetToken(kubeClient)
			if err != nil {
				return err
			}
		}
	} else {
		//Update the cluster role
		files := []string{
			"init/bootstrap_cluster_role.yaml",
		}
		out, err := apply.ApplyDirectly(clientHolder, reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
		if err != nil {
			return err
		}
		output = append(output, out...)
	}
	fmt.Printf("token: %s\n", token)

	fmt.Printf("please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		token,
		restConfig.Host,
	)

	return apply.WriteOutput(o.outputFile, output)
}

func waitForBootstrapSecret(kubeClient kubernetes.Interface, b wait.Backoff) (secret *corev1.Secret, err error) {
	err = retry.OnError(b, func(err error) bool {
		return err != nil
	}, func() error {
		secret, err = helpers.GetBootstrapSecretFromSA(kubeClient)
		return err
	})
	return
}

func (o *Options) applyToken(clientHolder *resourceapply.ClientHolder, reader *asset.ScenarioResourcesReader) ([]string, error) {
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
	out, err := apply.ApplyDirectly(clientHolder, reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return nil, err
	}
	return out, err
}
