// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"fmt"

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
	err := workClient.WorkV1().ManifestWorks(o.Cluster).Delete(context.TODO(), o.Workname, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		return err
	}

	// watch until work is fully removed
	err = helpers.WatchUntil(
		func() (watch.Interface, error) {
			return workClient.WorkV1().ManifestWorks(o.Cluster).Watch(context.TODO(), metav1.ListOptions{})
		},
		func(event watch.Event) bool {
			return event.Type == watch.Deleted
		},
	)
	if err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "work %s in cluster %s is deleted\n", o.Workname, o.Cluster)
	return nil
}
