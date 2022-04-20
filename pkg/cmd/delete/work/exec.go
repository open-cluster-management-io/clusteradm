// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
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
	if len(o.Cluster) == 0 {
		return fmt.Errorf("the name of the cluster must be specified")
	}

	return nil
}

func (o *Options) run() error {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	workClient, err := workclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return o.deleteWork(workClient)
}

func (o *Options) deleteWork(workClient *workclientset.Clientset) error {
	work, err := workClient.WorkV1().ManifestWorks(o.Cluster).Get(context.TODO(), o.Workname, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "work %s not found or is already deleted\n", o.Workname)
			return nil
		}
		return err
	}

	// start a goroutine to watch the delete event
	errChannel := make(chan error)
	timeout := 10 * time.Second
	go func(c chan<- error) {
		time.Sleep(timeout)
		c <- fmt.Errorf("delete work %s timeout, failed to delete", o.Workname)
	}(errChannel)

	go func(c chan<- error) {
		// watch until clusterset is removed
		e := helpers.WatchUntil(
			func() (watch.Interface, error) {
				return workClient.WorkV1().ManifestWorks(o.Cluster).Watch(context.TODO(), metav1.ListOptions{})
			},
			func(event watch.Event) bool {
				return event.Type == watch.Deleted
			},
		)
		c <- e

	}(errChannel)

	err = workClient.WorkV1().ManifestWorks(o.Cluster).Delete(context.TODO(), o.Workname, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	if o.Force {
		// check whether work is already deleted, if not, remove the finalizer
		work, err = workClient.WorkV1().ManifestWorks(o.Cluster).Get(context.TODO(), o.Workname, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Fprintf(o.Streams.Out, "work %s is deleted\n", o.Workname)
			return nil
		}

		if err != nil {
			return err
		}

		// if any finalizer exists, remove it.
		// if not, do nothing and wait for delete event.
		if len(work.ObjectMeta.Finalizers) != 0 {
			work.ObjectMeta.Finalizers = work.ObjectMeta.Finalizers[:0]

			_, err = workClient.WorkV1().ManifestWorks(o.Cluster).Update(context.TODO(), work, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// handle the error of watch function
	if err = <-errChannel; err != nil {
		close(errChannel)
		return err
	}

	fmt.Fprintf(o.Streams.Out, "work %s in cluster %s is deleted\n", o.Workname, o.Cluster)
	return nil
}
