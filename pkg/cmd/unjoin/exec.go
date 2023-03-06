// Copyright Contributors to the Open Cluster Management project
package unjoin

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	klusterletclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	appliedworkclient "open-cluster-management.io/api/client/work/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/check"
)

const (
	defaultKlusterletName       = "klusterlet"
	defaultAgentNamespace       = "open-cluster-management-agent"
	managedKubeconfigSecretName = "external-managed-kubeconfig"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("unjoin  options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, o.outputFile)

	o.values = Values{
		ClusterName:    o.clusterName,
		DeployMode:     operatorv1.InstallModeDefault,
		KlusterletName: defaultKlusterletName,
		AgentNamespace: defaultAgentNamespace,
	}
	klog.V(3).InfoS("values:", "clusterName", o.values.ClusterName)
	return nil
}

func (o *Options) validate() error {
	if o.values.ClusterName == "" {
		return fmt.Errorf("name is missing")
	}
	return nil
}

func (o *Options) run() error {
	// 1. get klusterlet cr by clustername
	// 2. check if any applied work still running
	// 3. delete klusterlet cr
	// 4. if --purge-operator=true and no klusterlet cr exists, purge the operator
	fmt.Fprintf(o.Streams.Out, "Remove applied resources in the managed cluster %s ... \n", o.clusterName)

	f := o.ClusteradmFlags.KubectlFactory
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}

	klusterletClient, err := klusterletclient.NewForConfig(config)
	if err != nil {
		return err
	}

	if err := check.CheckForKlusterletCRD(klusterletClient); err != nil {
		if errors.IsNotFound(err) {
			fmt.Println("klusterlet CRD not found, there is no need to unjoin.")
			return nil
		}
	}

	err = o.getKlusterlet(kubeClient, klusterletClient)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "klusterlet corresponds to %s not found", o.values.ClusterName)
			return nil
		}
		return err
	}

	appliedWorkClient, err := appliedworkclient.NewForConfig(config)
	if err != nil {
		return err
	}
	// In hosted mode, the work applied on managed cluster, so we should fetch managed cluster kubeconfig to build a work client
	if o.values.DeployMode == operatorv1.InstallModeHosted {
		kubeconfigSecret, err := kubeClient.CoreV1().Secrets(o.values.AgentNamespace).Get(context.Background(), managedKubeconfigSecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		kubeconfigBytes := kubeconfigSecret.Data["kubeconfig"]
		restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return err
		}
		appliedWorkClient, err = appliedworkclient.NewForConfig(restConfig)
		if err != nil {
			return err
		}
	}

	if isAppliedManifestWorkExist(appliedWorkClient) {
		fmt.Fprintf(o.Streams.Out, "appliedManifestWork still exist on the managed cluster, you should manually clean those works, uninstall kluster will cause those works out of control.")
		return nil
	}

	err = o.purgeKlusterlet(kubeClient, klusterletClient)
	if err != nil {
		return err
	}

	//Delete the other applied resources
	if o.purgeOperator {
		list, err := klusterletClient.OperatorV1().Klusterlets().List(context.Background(), metav1.ListOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		if len(list.Items) != 0 {
			fmt.Fprintf(o.Streams.Out, "operator not purged: there are other klusterlet on cluster\n")
			return nil
		}
		if err = purgeOperator(kubeClient, apiExtensionsClient); err != nil {
			return err
		}
	}

	fmt.Fprintf(o.Streams.Out, "Applied resources have been deleted during the %s joined stage. The status of mcl %s will be unknown in the hub cluster.\n", o.clusterName, o.clusterName)
	return nil

}

func (o *Options) getKlusterlet(kubeClient kubernetes.Interface, klusterletClient klusterletclient.Interface) error {
	list, err := klusterletClient.OperatorV1().Klusterlets().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		if item.Spec.ClusterName == o.values.ClusterName {
			if item.Spec.DeployOption.Mode == operatorv1.InstallModeHosted {
				o.values.DeployMode = item.Spec.DeployOption.Mode
				o.values.KlusterletName = item.Name
				o.values.AgentNamespace = o.values.KlusterletName
			}
			return nil
		}
	}

	return errors.NewNotFound(operatorv1.Resource("klusterlet"), o.values.ClusterName)
}

func isAppliedManifestWorkExist(client appliedworkclient.Interface) bool {
	obj, err := client.WorkV1().AppliedManifestWorks().List(context.Background(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return false
	}
	if err != nil {
		klog.Warningf("can not list applied manifest works: %v, ypu should check and delete the applied manifest works manully.", err)
		return false
	}
	if len(obj.Items) > 0 {
		return true
	}
	return false
}

func (o *Options) purgeKlusterlet(kubeClient kubernetes.Interface, klusterletClient klusterletclient.Interface) error {
	err := klusterletClient.OperatorV1().Klusterlets().Delete(context.Background(), o.values.KlusterletName, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		fmt.Fprintf(o.Streams.Out, "klusterlet %s is cleaned up already\n", o.values.KlusterletName)
		return nil
	}
	if err != nil {
		return err
	}

	b := retry.DefaultBackoff
	b.Duration = 1 * time.Second
	err = WaitResourceToBeDelete(context.Background(), klusterletClient, o.values.KlusterletName, b)
	if err != nil {
		return err
	}

	err = kubeClient.CoreV1().Namespaces().Delete(context.Background(), o.values.AgentNamespace, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		fmt.Fprintf(o.Streams.Out, "namespace %s is cleaned up already\n", o.values.AgentNamespace)
		return nil
	}
	if err != nil {
		return err
	}

	return nil

}

func purgeOperator(client kubernetes.Interface, extensionClient apiextensionsclient.Interface) error {
	var errs []error

	nameSpace := "open-cluster-management"
	err := client.AppsV1().
		Deployments(nameSpace).
		Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = extensionClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Delete(context.Background(), "klusterlets.operator.open-cluster-management.io", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoles().
		Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.RbacV1().
		ClusterRoleBindings().
		Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}
	err = client.CoreV1().
		ServiceAccounts("open-cluster-management").
		Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		errs = append(errs, err)
	}

	return utilerrors.NewAggregate(errs)
}

func WaitResourceToBeDelete(context context.Context, client klusterletclient.Interface, name string, b wait.Backoff) error {
	errGet := retry.OnError(b, func(err error) bool {
		return true
	}, func() error {
		_, err := client.OperatorV1().Klusterlets().Get(context, name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return nil
		}
		if err == nil {
			return fmt.Errorf("klusterlet still exists")
		}
		return err
	})
	return errGet

}
