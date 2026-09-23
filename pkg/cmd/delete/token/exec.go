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

func (o *Options) validate(ctx context.Context) error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}

	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsClusterManagerInstalled(ctx, apiExtensionsClient)
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("this is not a hub")
	}
	return err
}

func (o *Options) run(ctx context.Context) error {

	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}

	if o.ClusteradmFlags.DryRun {
		return nil
	}

	return o.deleteToken(ctx, kubeClient)
}

func (o *Options) deleteToken(ctx context.Context, kubeClient *kubernetes.Clientset) error {
	//Delete bootstrap token bindings
	err := kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, config.BootstrapClusterRoleBindingName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, config.BootstrapClusterRoleBindingSAName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	//Delete Roles
	err = kubeClient.RbacV1().ClusterRoles().Delete(ctx, config.BootstrapClusterRoleName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	//Detele bootstrap token secret
	secret, err := helpers.GetBootstrapSecret(ctx, kubeClient)
	if err == nil {
		err = kubeClient.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	//Delete service account
	err = kubeClient.CoreV1().ServiceAccounts(config.OpenClusterManagementNamespace).Delete(ctx, config.BootstrapSAName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	//No need to delete the secret containing the token
	//as it will be automatically deleted because the SA is deleted
	return nil
}
