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
	"k8s.io/client-go/util/retry"

	klusterletclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	appliedworkclient "open-cluster-management.io/api/client/work/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"
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
	nameSpace := "open-cluster-management"
	output := make([]string, 0)
	fmt.Printf("Remove applied resources in the managed cluster %s ... \n", o.clusterName)

	//Delete Klusterlet CR resources firstly
	f := o.ClusteradmFlags.KubectlFactory
	config, err := f.ToRESTConfig()
	if err != nil {
		log.Fatal(err)
	}
	kubeClient, apiExtensionsClient, _, err := helpers.GetClients(f)
	if err != nil {
		log.Fatal(err)
	}
	appliedWorkClient, err := appliedworkclient.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	if IsAppliedManifestWorkExist(appliedWorkClient) {

		log.Fatal("AppliedManifestWork exist on the managed cluster, uninstalling the klusterlet will cause that the manifestworks on hub cannot be cleaned.")
	} else {
		//Create klusterlet client
		klusterletClient, err := klusterletclient.NewForConfig(config)
		if err != nil {
			log.Fatal(err)
		}
		err = klusterletClient.OperatorV1().Klusterlets().Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Fatal(err)
		}
		if !o.ClusteradmFlags.DryRun {
			b := retry.DefaultBackoff
			b.Duration = 1 * time.Second

			err = WaitResourceToBeDelete(context.Background(), klusterletClient, "klusterlet", b)
			if err != nil && !errors.IsNotFound(err) {
				log.Fatal("Resource Klusterlet should be deleted first:", err)
			}
		}
	}

	//Delete the other applied resources
	if o.purgeOperator {
		if !o.ClusteradmFlags.DryRun {
			_ = kubeClient.AppsV1().
				Deployments(nameSpace).
				Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
			_ = apiExtensionsClient.ApiextensionsV1().
				CustomResourceDefinitions().
				Delete(context.Background(), "klusterlets.operator.open-cluster-management.io", metav1.DeleteOptions{})
			_ = kubeClient.CoreV1().
				Namespaces().
				Delete(context.Background(), "open-cluster-management-agent", metav1.DeleteOptions{})
			_ = kubeClient.RbacV1().
				ClusterRoles().
				Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
			_ = kubeClient.RbacV1().
				ClusterRoleBindings().
				Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
			_ = kubeClient.CoreV1().
				ServiceAccounts("open-cluster-management").
				Delete(context.Background(), "klusterlet", metav1.DeleteOptions{})
		}
		log.Println("Other resources have been deleted.")
	}

	fmt.Printf("Applied resources have been deleted during the %s joined stage. The status of mcl %s will be unknown in the hub cluster.\n", o.clusterName, o.clusterName)
	return apply.WriteOutput(o.outputFile, output)

}

func WaitResourceToBeDelete(context context.Context, client klusterletclient.Interface, name string, b wait.Backoff) error {

	errGet := retry.OnError(b, func(err error) bool {
		if err != nil && !errors.IsNotFound(err) {
			log.Println("Wait to deleted resource klusterlet:", err)
			return true
		}
		return false
	}, func() error {
		_, err := client.OperatorV1().Klusterlets().Get(context, name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("klusterlet is still exist")
		}
		return err
	})
	return errGet

}
func IsAppliedManifestWorkExist(client appliedworkclient.Interface) bool {
	obj, err := client.WorkV1().AppliedManifestWorks().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	if len(obj.Items) > 0 {
		return true
	}
	return false
}
