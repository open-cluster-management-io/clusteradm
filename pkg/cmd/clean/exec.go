// Copyright Contributors to the Open Cluster Management project

package init

import (
	"context"
	"fmt"
	"log"
	"time"

	clustermanagerclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("clean options:", "dry-run", o.ClusteradmFlags.DryRun, "output-file", o.outputFile)
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
		return fmt.Errorf("hub not be initialized")
	}
	return nil
}

func (o *Options) run() error {
	output := make([]string, 0)

	//Clean ClusterManager CR resource firstly
	f := o.ClusteradmFlags.KubectlFactory
	config, err := f.ToRESTConfig()
	if err != nil {
		log.Fatal(err)
	}
	clusterManagerClient, err := clustermanagerclient.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	if IsClusterManagerExist(clusterManagerClient) {
		err = clusterManagerClient.OperatorV1().ClusterManagers().Delete(context.Background(), o.clusterManageName, metav1.DeleteOptions{})
		if err != nil {
			log.Fatal(err)
		}
		if !o.ClusteradmFlags.DryRun {
			b := retry.DefaultBackoff
			b.Duration = 1 * time.Second

			err = WaitResourceToBeDelete(context.Background(), clusterManagerClient, o.clusterManageName, b)
			if !errors.IsNotFound(err) {
				log.Fatal("Cluster Manager resource should be deleted firstly.")
			}
		}
	}
	//Clean other resources
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}
	err = kubeClient.AppsV1().Deployments("open-cluster-management").Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), "clustermanagers.operator.open-cluster-management.io", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = kubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = kubeClient.CoreV1().ServiceAccounts("open-cluster-management").Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	if o.useBootstrapToken {
		err = kubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "system:open-cluster-management:bootstrap", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "cluster-bootstrap", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		err = kubeClient.CoreV1().Secrets("kube-system").Delete(context.Background(), "bootstrap-token-"+o.values.Hub.TokenID, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	} else {
		err = kubeClient.RbacV1().ClusterRoles().Delete(context.Background(), "system:open-cluster-management:bootstrap", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.Background(), "cluster-bootstrap-sa", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		err = kubeClient.CoreV1().ServiceAccounts("open-cluster-management").Delete(context.Background(), "cluster-bootstrap", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	err = kubeClient.CoreV1().Namespaces().Delete(context.Background(), "open-cluster-management", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	fmt.Println("The multicluster hub control plane has been clean up successfully!")

	return apply.WriteOutput(o.outputFile, output)
}
func WaitResourceToBeDelete(context context.Context, client clustermanagerclient.Interface, name string, b wait.Backoff) error {

	errGet := retry.OnError(b, func(err error) bool {
		if err != nil && !errors.IsNotFound(err) {
			log.Printf("Wait to delete cluster manager resource: %s.\n", name)
			return true
		}
		return false
	}, func() error {
		_, err := client.OperatorV1().ClusterManagers().Get(context, name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("ClusterManager is still exist")
		}
		return err
	})
	return errGet

}
func IsClusterManagerExist(cilent clustermanagerclient.Interface) bool {
	obj, err := cilent.OperatorV1().ClusterManagers().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	if len(obj.Items) > 0 {
		return true
	}
	return false
}
