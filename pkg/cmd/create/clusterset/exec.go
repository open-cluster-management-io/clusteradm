// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterapiv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.Clustersets = args

	return nil
}

func (o *Options) Validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Clustersets) == 0 {
		return fmt.Errorf("the name of the clusterset must be specified")
	}
	if len(o.Clustersets) > 1 {
		return fmt.Errorf("only one clusterset can be created")
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

	clusterSetName := o.Clustersets[0]

	return o.runWithClient(clusterClient, o.ClusteradmFlags.DryRun, clusterSetName)
}

func (o *Options) runWithClient(clusterClient clusterclientset.Interface,
	dryRun bool,
	clusterset string) error {

	_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.TODO(), clusterset, metav1.GetOptions{})
	if err == nil {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s is already created\n", clusterset)
		return nil
	}

	if dryRun {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s is created\n", clusterset)
		return nil
	}

	mcs := &clusterapiv1beta2.ManagedClusterSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterset,
		},
	}

	_, err = clusterClient.ClusterV1beta2().ManagedClusterSets().Create(context.TODO(), mcs, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "Clusterset %s is created\n", clusterset)
	return nil
}
