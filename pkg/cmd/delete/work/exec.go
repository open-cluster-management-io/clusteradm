// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"errors"
	"fmt"
	"time"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	helperwait "open-cluster-management.io/clusteradm/pkg/helpers/wait"
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

	printDeleteMsg := func() {
		fmt.Fprintf(o.Streams.Out, "work %s in cluster %s is deleted\n", o.Workname, cluster)
	}

	err = workClient.WorkV1().ManifestWorks(cluster).Delete(ctx, o.Workname, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	if k8serrors.IsNotFound(err) {
		printDeleteMsg()
		return nil
	}

	if o.Force {
		// check whether work is already deleted, if not, remove the finalizer
		work, err := workClient.WorkV1().ManifestWorks(cluster).Get(ctx, o.Workname, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			printDeleteMsg()
			return nil
		}

		if err != nil {
			return err
		}

		// if any finalizer exists, remove it.
		// if not, do nothing and wait for the work to be deleted.
		if len(work.Finalizers) != 0 {
			work.Finalizers = work.Finalizers[:0]

			_, err = workClient.WorkV1().ManifestWorks(cluster).Update(ctx, work, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	err = wait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(o.ClusteradmFlags.Timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := workClient.WorkV1().ManifestWorks(cluster).Get(ctx, o.Workname, metav1.GetOptions{})
		if helperwait.IsFatalAPIError(err) {
			return false, err
		}
		return k8serrors.IsNotFound(err), nil
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("delete work %s timeout, failed to delete", o.Workname)
		}
		return err
	}

	printDeleteMsg()
	return nil
}
