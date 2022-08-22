// Copyright Contributors to the Open Cluster Management project
package unjoin

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	klusterletclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	appliedworkclient "open-cluster-management.io/api/client/work/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("unjoin  options:", "dry-run", o.ClusteradmFlags.DryRun, "cluster", o.clusterName, o.outputFile)

	o.values = Values{
		ClusterName: o.clusterName,
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

	// Delete the applied resource in the Managed cluster
	fmt.Fprintf(o.Streams.Out, "Remove applied resources in the managed cluster %s ... \n", o.clusterName)

	//Delete Klusterlet CR resources firstly
	f := o.ClusteradmFlags.KubectlFactory
	config, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}
	appliedWorkClient, err := appliedworkclient.NewForConfig(config)
	if err != nil {
		return err
	}

	if IsAppliedManifestWorkExist(appliedWorkClient) {
		return fmt.Errorf("appliedManifestWork exist on the managed cluster, uninstalling the klusterlet will cause that the manifestworks on hub cannot be cleaned")
	} else {
		//Create klusterlet client
		klusterletClient, err := klusterletclient.NewForConfig(config)
		if err != nil {
			return err
		}
		err = klusterletClient.OperatorV1().Klusterlets().Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
		if errors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "klusterlet is cleaned up already")
			return nil
		}
		if err != nil {
			return err
		}
		b := retry.DefaultBackoff
		b.Duration = 1 * time.Second

		err = WaitResourceToBeDelete(context.Background(), klusterletClient, "klusterlet", b)
		if err != nil {
			return err
		}
	}

	//Delete the other applied resources
	if o.purgeOperator {
		if err := puregeOperator(kubeClient, apiExtensionsClient); err != nil {
			return err
		}
	}

	fmt.Fprintf(o.Streams.Out, "Applied resources have been deleted during the %s joined stage. The status of mcl %s will be unknown in the hub cluster.\n", o.clusterName, o.clusterName)
	return nil

}

func puregeOperator(client kubernetes.Interface, extensionClient apiextensionsclient.Interface) error {
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

func IsAppliedManifestWorkExist(client appliedworkclient.Interface) bool {
	obj, err := client.WorkV1().AppliedManifestWorks().List(context.Background(), metav1.ListOptions{})
	if errors.IsNotFound(err) {
		return false
	}
	if err != nil {
		log.Fatal(err)
	}
	if len(obj.Items) > 0 {
		return true
	}
	return false
}
