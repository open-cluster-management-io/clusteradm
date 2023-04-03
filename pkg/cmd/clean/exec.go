// Copyright Contributors to the Open Cluster Management project

package init

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	clustermanagerclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("clean options:", "dry-run", o.ClusteradmFlags.DryRun, "output-file", o.OutputFile)
	o.Values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
		},
	}
	return nil
}

func (o *Options) Validate() error {
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
		return fmt.Errorf("hub has not been initialized")
	}
	return nil
}

func (o *Options) Run() error {
	//Clean ClusterManager CR resource firstly
	f := o.ClusteradmFlags.KubectlFactory
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterManagerClient, err := clustermanagerclient.NewForConfig(config)
	if err != nil {
		return err
	}

	//Clean other resources
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}

	if err := o.removeBootStrapSecret(kubeClient); err != nil {
		return err
	}

	err = clusterManagerClient.OperatorV1().ClusterManagers().Delete(context.Background(), o.ClusterManageName, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		fmt.Fprintf(o.Streams.Out, "The multicluster hub control plane is cleand up already\n")
		return nil
	}
	b := retry.DefaultBackoff
	b.Duration = 3 * time.Second

	err = WaitResourceToBeDelete(context.Background(), clusterManagerClient, o.ClusterManageName, b)
	if err != nil {
		return err
	}

	if o.purgeOperator {
		if err := puregeOperator(kubeClient, apiExtensionsClient); err != nil {
			return err
		}
	}

	fmt.Fprintf(o.Streams.Out, "The multicluster hub control plane has been clean up successfully!\n")

	return nil
}
func WaitResourceToBeDelete(context context.Context, client clustermanagerclient.Interface, name string, b wait.Backoff) error {
	errGet := retry.OnError(b, func(err error) bool {
		return true
	}, func() error {
		_, err := client.OperatorV1().ClusterManagers().Get(context, name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		if err == nil {
			return fmt.Errorf("cluster manager still exists")
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

func (o *Options) removeBootStrapSecret(client kubernetes.Interface) error {
	var errs []error
	err := client.RbacV1().
		ClusterRoles().
		Delete(context.Background(), "system:open-cluster-management:bootstrap", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoleBindings().
		Delete(context.Background(), "cluster-bootstrap", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	listOpts := metav1.ListOptions{LabelSelector: "app=cluster-manager"}
	err = client.CoreV1().
		Secrets("kube-system").
		DeleteCollection(context.Background(), metav1.DeleteOptions{}, listOpts)
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoleBindings().
		Delete(context.Background(), "cluster-bootstrap-sa", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.CoreV1().
		ServiceAccounts("open-cluster-management").
		Delete(context.Background(), "cluster-bootstrap", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	return utilerrors.NewAggregate(errs)
}

func puregeOperator(client kubernetes.Interface, extensionClient apiextensionsclient.Interface) error {
	var errs []error
	err := client.AppsV1().
		Deployments("open-cluster-management").
		Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = extensionClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Delete(context.Background(), "clustermanagers.operator.open-cluster-management.io", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoles().
		Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoleBindings().
		Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.CoreV1().
		ServiceAccounts("open-cluster-management").
		Delete(context.Background(), "cluster-manager", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.CoreV1().
		Namespaces().
		Delete(context.Background(), "open-cluster-management", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}

	return utilerrors.NewAggregate(errs)
}
