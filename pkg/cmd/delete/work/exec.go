// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"errors"
	"fmt"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

func (o *Options) complete(_ *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("work name must be specified")
	}

	if len(args) > 1 {
		return fmt.Errorf("only one work name can be specified")
	}

	o.Workname = args[0]

	return nil
}

func (o *Options) validate() error {

	if err := o.ClusteradmFlags.ValidateHub(); err != nil {
		return err
	}

	if err := o.ClusterOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *Options) run(ctx context.Context) error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	workClient, err := workclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	var errs []error
	for cluster := range o.ClusterOptions.AllClusters() {
		err := o.deleteWork(ctx, workClient, cluster)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func (o *Options) deleteWork(ctx context.Context, workClient *workclientset.Clientset, cluster string) error {
	_, err := workClient.WorkV1().ManifestWorks(cluster).Get(ctx, o.Workname, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "work %s not found or is already deleted\n", o.Workname)
			return nil
		}
		return err
	}

	watchCtx, cancel := context.WithTimeout(ctx, time.Duration(o.ClusteradmFlags.Timeout)*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- helpers.WatchUntil(
			watchCtx,
			func() (watch.Interface, error) {
				return workClient.WorkV1().ManifestWorks(cluster).Watch(watchCtx, metav1.ListOptions{})
			},
			func(event watch.Event) bool {
				return event.Type == watch.Deleted
			},
		)
	}()

	err = workClient.WorkV1().ManifestWorks(cluster).Delete(ctx, o.Workname, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if o.Force {
		// check whether work is already deleted, if not, remove the finalizer
		work, err := workClient.WorkV1().ManifestWorks(cluster).Get(ctx, o.Workname, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "work %s is deleted\n", o.Workname)
			return nil
		}

		if err != nil {
			return err
		}

		// if any finalizer exists, remove it.
		// if not, do nothing and wait for delete event.
		if len(work.Finalizers) != 0 {
			work.Finalizers = work.Finalizers[:0]

			_, err = workClient.WorkV1().ManifestWorks(cluster).Update(ctx, work, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	if err = <-errCh; err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete work %s timeout, failed to delete", o.Workname)
		}
		return err
	}

	fmt.Fprintf(o.Streams.Out, "work %s in cluster %s is deleted\n", o.Workname, cluster)
	return nil
}
