// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	helperwait "open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

func (o *Options) complete(_ *cobra.Command, args []string) (err error) {

	o.Clustersets = args

	return nil
}

func (o *Options) validate() (err error) {
	err = o.ClusteradmFlags.ValidateHub()
	if err != nil {
		return err
	}

	if len(o.Clustersets) == 0 {
		return fmt.Errorf("the name of the clusterset must be specified")
	}
	if len(o.Clustersets) > 1 {
		return fmt.Errorf("only one clusterset can be deleted")
	}

	return nil
}

func (o *Options) run(ctx context.Context) (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	clusterSetName := o.Clustersets[0]

	return o.runWithClient(ctx, clusterClient, o.ClusteradmFlags.DryRun, clusterSetName)
}

// check unband first

func (o *Options) runWithClient(ctx context.Context, clusterClient clusterclientset.Interface,
	dryRun bool,
	clusterset string) error {

	// not allow to delete default clusterset
	if clusterset == "default" {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s can not be deleted\n", clusterset)
		return nil
	}

	// check existing
	_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(ctx, clusterset, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "Clusterset %s not found or is already deleted\n", clusterset)
			return nil
		}
		return err
	}

	// check binding
	list, err := clusterClient.ClusterV1beta2().ManagedClusterSetBindings(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", clusterset),
	})
	// if exist, return
	if err == nil && len(list.Items) != 0 {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s still bind to a namespace! Please unbind before deleted.\n", clusterset)
		return nil
	}
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if dryRun {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s is deleted\n", clusterset)
		return nil
	}

	// delete
	err = clusterClient.ClusterV1beta2().ManagedClusterSets().Delete(ctx, clusterset, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = wait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(o.ClusteradmFlags.Timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(ctx, clusterset, metav1.GetOptions{})
		if helperwait.IsFatalAPIError(err) {
			return false, err
		}
		return k8serrors.IsNotFound(err), nil
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete clusterset %s timeout, failed to delete", clusterset)
		}
		return err
	}

	fmt.Fprintf(o.Streams.Out, "Clusterset %s is deleted\n", clusterset)
	return nil
}
