// Copyright Contributors to the Open Cluster Management project
package unbind

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("the name of the clusterset must be specified")
	}

	if len(args) > 1 {
		return fmt.Errorf("only one clusterset can be specified")
	}

	o.Clusterset = args[0]

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Namespace) == 0 {
		return fmt.Errorf("namespace name must be specified in --namespace")
	}

	return nil
}

func (o *Options) Run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	_, err = clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.TODO(), o.Clusterset, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = clusterClient.ClusterV1beta2().ManagedClusterSetBindings(o.Namespace).Delete(context.TODO(), o.Clusterset, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "Clusterset %s is unbind from namespace %s\n", o.Clusterset, o.Namespace)
	return nil
}
