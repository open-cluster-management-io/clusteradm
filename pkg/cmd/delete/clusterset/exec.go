// Copyright Contributors to the Open Cluster Management project
package clusterset

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
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
		if errors.IsNotFound(err) {
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
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if dryRun {
		fmt.Fprintf(o.Streams.Out, "Clusterset %s is deleted\n", clusterset)
		return nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- helpers.WatchUntil(
			ctx,
			func() (watch.Interface, error) {
				return clusterClient.ClusterV1beta2().ManagedClusterSets().Watch(ctx, metav1.ListOptions{
					FieldSelector: fmt.Sprintf("metadata.name=%s", clusterset),
				})
			},
			func(event watch.Event) bool {
				return event.Type == watch.Deleted
			},
		)
	}()

	// delete
	err = clusterClient.ClusterV1beta2().ManagedClusterSets().Delete(ctx, clusterset, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	if err = <-errCh; err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "Clusterset %s is deleted\n", clusterset)
	return nil
}
