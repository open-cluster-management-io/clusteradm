// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	workapiv1 "open-cluster-management.io/api/work/v1"
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

func (o *Options) validate() (err error) {
	if len(o.Cluster) == 0 {
		return fmt.Errorf("the name of the cluster must be specified")
	}
	if len(*o.FileNameFlags.Filenames) == 0 {
		return fmt.Errorf("manifest files must be specified")
	}
	return nil
}

func (o *Options) run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	workClient, err := workclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	manifests, err := o.readManifests()
	if err != nil {
		return err
	}

	err = o.applyWork(workClient, manifests)
	if err != nil {
		return err
	}

	fmt.Fprintf(o.Streams.Out, "work %s in cluster %s is created\n", o.Workname, o.Cluster)
	return
}

func (o *Options) readManifests() ([]workapiv1.Manifest, error) {
	opt := o.FileNameFlags.ToOptions()
	builder := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false, &opt).
		Flatten().
		ContinueOnError()
	result := builder.Do()

	if err := result.Err(); err != nil {
		return nil, err
	}

	manifests := []workapiv1.Manifest{}

	items, err := result.Infos()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		manifests = append(manifests, workapiv1.Manifest{RawExtension: runtime.RawExtension{Object: item.Object}})
	}

	return manifests, nil
}

func (o *Options) applyWork(workClient workclientset.Interface, manifests []workapiv1.Manifest) error {
	work, err := workClient.WorkV1().ManifestWorks(o.Cluster).Get(context.TODO(), o.Workname, metav1.GetOptions{})

	switch {
	case errors.IsNotFound(err):
		work = &workapiv1.ManifestWork{
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.Workname,
				Namespace: o.Cluster,
			},
			Spec: workapiv1.ManifestWorkSpec{
				Workload: workapiv1.ManifestsTemplate{
					Manifests: manifests,
				},
			},
		}
		_, createErr := workClient.WorkV1().ManifestWorks(o.Cluster).Create(context.TODO(), work, metav1.CreateOptions{})
		return createErr
	case err != nil:
		return err
	}

	if !o.Overwrite {
		return fmt.Errorf("work %s in cluster %s already exists", o.Workname, o.Cluster)
	}

	work.Spec.Workload.Manifests = manifests
	_, err = workClient.WorkV1().ManifestWorks(o.Cluster).Update(context.TODO(), work, metav1.UpdateOptions{})
	return err
}
