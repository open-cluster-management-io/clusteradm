// Copyright Contributors to the Open Cluster Management project
package token

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
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

	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}

	if o.ClusteradmFlags.DryRun {
		return nil
	}

	return o.deleteToken(kubeClient)
}

func (o *Options) deleteToken(kubeClient *kubernetes.Clientset) error {
	//Delete bootstrap token bindings
	err := kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), config.BootstrapClusterRoleBindingName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), config.BootstrapClusterRoleBindingSAName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	//Delete Roles
	err = kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), config.BootstrapClusterRoleName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	//Detele bootstrap token secret
	secret, err := helpers.GetBootstrapSecret(context.TODO(), kubeClient)
	if err == nil {
		err = kubeClient.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	//Delete service account
	err = kubeClient.CoreV1().ServiceAccounts(config.OpenClusterManagementNamespace).Delete(context.TODO(), config.BootstrapSAName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	//No need to delete the secret containing the token
	//as it will be automatically deleted because the SA is deleted
	return nil
}
