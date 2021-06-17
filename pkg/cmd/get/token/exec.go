// Copyright Contributors to the Open Cluster Management project
package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

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
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}
	//Search accross all ns as the cluster-manager is not always installed in the same ns
	l, err := kubeClient.CoreV1().
		Pods("").
		List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%v = %v", config.LabelApp, config.LabelAppClusterApp)})
	if err != nil {
		return err
	}
	if len(l.Items) == 0 {
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

	token, err := getToken(kubeClient)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		files := []string{
			"init/namespace.yaml",
		}
		if o.useBootstrapToken {
			files = append(files,
				"init/namespace.yaml",
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
			return err
		}
		output = append(output, out...)
		//Make sure that the sa token is ready
		if !o.useBootstrapToken {
			b := retry.DefaultBackoff
			b.Duration = 100 * time.Millisecond
			_, err = waitForBootstrapSecret(kubeClient, b)
			if err != nil {
				return err
			}
		}
		token, err = getToken(kubeClient)
		if err != nil {
			return err
		}
	}
	fmt.Printf("token=%s\n", token)

	fmt.Printf("please log on spoke and run:\n%s join --hub-token %s --hub-apiserver %s --cluster-name <cluster_name>\n",
		helpers.GetExampleHeader(),
		token,
		restConfig.Host,
	)

	return apply.WriteOutput("", output)
}

func waitForBootstrapSecret(kubeClient kubernetes.Interface, b wait.Backoff) (secret *corev1.Secret, err error) {
	err = retry.OnError(b, func(err error) bool {
		return err != nil
	}, func() error {
		secret, err = helpers.GetBootstrapSecret(kubeClient)
		return err
	})
	return
}

func getToken(kubeClient kubernetes.Interface) (string, error) {
	var bootstrapSecret *corev1.Secret
	saSecret, err := helpers.GetBootstrapSecret(kubeClient)
	if err != nil {
		if errors.IsNotFound(err) {
			//As no SA search for bootstrap token
			l, err := kubeClient.CoreV1().
				Secrets("kube-system").
				List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%v = %v", config.LabelApp, config.LabelAppClusterApp)})
			if err != nil {
				return "", err
			}
			for _, s := range l.Items {
				if strings.HasPrefix(s.Name, config.BootstrapSecretPrefix) {
					bootstrapSecret = &s
				}
			}
			if bootstrapSecret != nil {
				return fmt.Sprintf("%s.%s", string(bootstrapSecret.Data["token-id"]), string(bootstrapSecret.Data["token-secret"])), nil
			}
		}
		return "", err
	} else {
		return string(saSecret.Data["token"]), nil
	}
}
