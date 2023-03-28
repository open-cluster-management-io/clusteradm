// Copyright Contributors to the Open Cluster Management project
package work

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/resource"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclientset "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
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

func (o *Options) validate() error {

	if err := o.ClusteradmFlags.ValidateHub(); err != nil {
		return err
	}
	if err := o.ClusterOption.Validate(); err != nil {
		return err
	}

	clusters := o.ClusterOption.AllClusters()
	if clusters.Len() == 0 && len(o.Placement) == 0 {
		return fmt.Errorf("--clusters or --placement must be specified")
	}
	if clusters.Len() > 0 && len(o.Placement) > 0 {
		return fmt.Errorf("--clusters and --placement can only specify one")
	}
	if len(o.Placement) > 0 && len(strings.Split(o.Placement, "/")) != 2 {
		return fmt.Errorf("the name of the placement %s must be in the format of <namespace>/<name>", o.Placement)
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
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	manifests, err := o.readManifests()
	if err != nil {
		return err
	}

	addedClusters, deletedClusters, err := o.getClusters(workClient, clusterClient)
	if err != nil {
		return err
	}

	err = o.applyWork(workClient, manifests, addedClusters, deletedClusters)
	if err != nil {
		return err
	}

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

func (o *Options) getPlacement(clusterClient *clusterclientset.Clientset) (*clusterv1beta1.Placement, error) {
	parts := strings.Split(o.Placement, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("the name of the placement %s must be in the format of <namespace>/<name>", o.Placement)
	}

	namespace, name := parts[0], parts[1]
	placement, err := clusterClient.ClusterV1beta1().Placements(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get placement %s", err)
	}

	return placement, nil
}

func (o *Options) getWorkDepolyClusters(workClient workclientset.Interface) (sets.String, error) {
	works, err := workClient.WorkV1().ManifestWorks("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	depolyClusters := sets.NewString()
	for _, work := range works.Items {
		if work.Name == o.Workname {
			depolyClusters.Insert(work.Namespace)
		}
	}
	return depolyClusters, nil
}

func (o *Options) getClusters(workClient workclientset.Interface, clusterClient *clusterclientset.Clientset) (sets.String, sets.String, error) {
	existingDeployClusters, err := o.getWorkDepolyClusters(workClient)
	if err != nil {
		return nil, nil, err
	}

	// if define --clusters, return that as addedClusters and no deletedClusters
	clusters := o.ClusterOption.AllClusters()
	if clusters.Len() > 0 {
		return clusters, nil, nil
	}

	placement, err := o.getPlacement(clusterClient)
	if err != nil {
		return nil, nil, err
	}

	pdtracker := clusterv1beta1.NewPlacementDecisionClustersTracker(placement, placementDecisionGetter{clusterClient: clusterClient}, existingDeployClusters)
	addedClusters, deletedClusters, err := pdtracker.Get()
	if err != nil {
		return nil, nil, err
	}

	return addedClusters, deletedClusters, nil
}

func (o *Options) applyWork(workClient workclientset.Interface, manifests []workapiv1.Manifest, addedClusters, deletedClusters sets.String) error {
	for clusterName := range deletedClusters {
		if o.Overwrite {
			if err := workClient.WorkV1().ManifestWorks(clusterName).Delete(context.TODO(), o.Workname, metav1.DeleteOptions{}); err != nil {
				fmt.Fprintf(o.Streams.Out, "failed to delete work %s in cluster %s as %s\n", o.Workname, clusterName, err)
			}
			fmt.Fprintf(o.Streams.Out, "delete work %s in cluster %s\n", o.Workname, clusterName)
		}
	}

	for clusterName := range addedClusters {
		work, err := workClient.WorkV1().ManifestWorks(clusterName).Get(context.TODO(), o.Workname, metav1.GetOptions{})

		switch {
		case errors.IsNotFound(err):
			work = &workapiv1.ManifestWork{
				ObjectMeta: metav1.ObjectMeta{
					Name:      o.Workname,
					Namespace: clusterName,
				},
				Spec: workapiv1.ManifestWorkSpec{
					Workload: workapiv1.ManifestsTemplate{
						Manifests: manifests,
					},
				},
			}
			if _, err := workClient.WorkV1().ManifestWorks(clusterName).Create(context.TODO(), work, metav1.CreateOptions{}); err != nil {
				return err
			}
			fmt.Fprintf(o.Streams.Out, "create work %s in cluster %s\n", o.Workname, clusterName)
			continue
		case err != nil:
			return err
		}

		if !o.Overwrite {
			fmt.Fprintf(o.Streams.Out, "work %s in cluster %s already exists\n", o.Workname, clusterName)
		} else {
			work.Spec.Workload.Manifests = manifests
			if _, err := workClient.WorkV1().ManifestWorks(clusterName).Update(context.TODO(), work, metav1.UpdateOptions{}); err != nil {
				return err
			}
			fmt.Fprintf(o.Streams.Out, "update work %s in cluster %s\n", o.Workname, clusterName)
		}
	}

	return nil
}

type placementDecisionGetter struct {
	clusterClient *clusterclientset.Clientset
}

func (pdl placementDecisionGetter) List(selector labels.Selector, namespace string) ([]*clusterv1beta1.PlacementDecision, error) {
	decisionList, err := pdl.clusterClient.ClusterV1beta1().PlacementDecisions(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	var decisions []*clusterv1beta1.PlacementDecision
	for i := range decisionList.Items {
		decisions = append(decisions, &decisionList.Items[i])
	}
	return decisions, nil
}
