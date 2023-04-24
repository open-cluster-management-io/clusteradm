// Copyright Contributors to the Open Cluster Management project
package placement

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"strconv"
	"strings"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if len(args) == 0 {
		return fmt.Errorf("placement name must be specified")
	}

	if len(args) > 1 {
		return fmt.Errorf("only one placement name can be specified")
	}

	o.Placement = args[0]

	return nil
}

func (o *Options) validate() error {

	if err := o.ClusteradmFlags.ValidateHub(); err != nil {
		return err
	}

	return nil
}

func (o *Options) run() (err error) {
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	desiredPlacement := &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Placement,
			Namespace: o.Namespace,
		},
	}

	if len(o.ClusterSets) > 0 {
		for _, clusterset := range o.ClusterSets {
			_, err := clusterClient.ClusterV1beta2().ManagedClusterSetBindings(o.Namespace).Get(context.TODO(), clusterset, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		desiredPlacement.Spec.ClusterSets = o.ClusterSets
	}

	if o.NumOfClusters > 0 {
		desiredPlacement.Spec.NumberOfClusters = pointer.Int32(o.NumOfClusters)
	}

	if len(o.ClusterSelector) > 0 {
		desiredPlacement.Spec.Predicates = []clusterv1beta1.ClusterPredicate{}
		for _, s := range o.ClusterSelector {
			selector, err := metav1.ParseToLabelSelector(s)
			if err != nil {
				return fmt.Errorf("failed to parse selector %s: %v", s, err)
			}
			desiredPlacement.Spec.Predicates = append(desiredPlacement.Spec.Predicates, clusterv1beta1.ClusterPredicate{
				RequiredClusterSelector: clusterv1beta1.ClusterSelector{
					LabelSelector: *selector,
				},
			})
		}
	}

	if len(o.Prioritizers) > 0 {
		desiredPlacement.Spec.PrioritizerPolicy = clusterv1beta1.PrioritizerPolicy{
			Mode:           clusterv1beta1.PrioritizerPolicyModeAdditive,
			Configurations: []clusterv1beta1.PrioritizerConfig{},
		}
		for _, p := range o.Prioritizers {
			config, err := parsePrioritizer(p)
			if err != nil {
				return err
			}
			desiredPlacement.Spec.PrioritizerPolicy.Configurations = append(desiredPlacement.Spec.PrioritizerPolicy.Configurations, *config)
		}
	}

	return o.applyPlacement(clusterClient, desiredPlacement)
}

func parsePrioritizer(s string) (*clusterv1beta1.PrioritizerConfig, error) {
	ps := strings.Split(s, ":")
	if len(ps) < 3 {
		return nil, fmt.Errorf("prioritizer %s format is not correct", s)
	}
	switch ps[0] {
	case clusterv1beta1.ScoreCoordinateTypeBuiltIn:
		if len(ps) != 3 {
			return nil, fmt.Errorf("prioritizer %s format is not correct, should be Builtin:{Type}:{Weight}", s)
		}
		weight, err := strconv.ParseInt(ps[2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("weight in prioritizer %s is not correct: %v", s, err)
		}
		return &clusterv1beta1.PrioritizerConfig{
			ScoreCoordinate: &clusterv1beta1.ScoreCoordinate{
				Type:    clusterv1beta1.ScoreCoordinateTypeBuiltIn,
				BuiltIn: ps[1],
			},
			Weight: int32(weight),
		}, nil
	case clusterv1beta1.ScoreCoordinateTypeAddOn:
		if len(ps) != 4 {
			return nil, fmt.Errorf("prioritizer %s format is not correct, should be Addon:{Type}:{ScoreName}:{Weight}", s)
		}
		weight, err := strconv.ParseInt(ps[3], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("weight in prioritizer %s is not correct: %v", s, err)
		}
		return &clusterv1beta1.PrioritizerConfig{
			ScoreCoordinate: &clusterv1beta1.ScoreCoordinate{
				Type: clusterv1beta1.ScoreCoordinateTypeAddOn,
				AddOn: &clusterv1beta1.AddOnScore{
					ResourceName: ps[1],
					ScoreName:    ps[2],
				},
			},
			Weight: int32(weight),
		}, nil
	}

	return nil, fmt.Errorf("unkown prioritizer type %s for %s", ps[0], s)
}

func (o *Options) applyPlacement(clusterClient clusterclientset.Interface, placement *clusterv1beta1.Placement) error {
	placementOrigin, err := clusterClient.ClusterV1beta1().Placements(o.Namespace).Get(context.TODO(), placement.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, createErr := clusterClient.ClusterV1beta1().Placements(o.Namespace).Create(context.TODO(), placement, metav1.CreateOptions{})
		return createErr
	}
	if err != nil {
		return err
	}

	if !o.Overwrite {
		return fmt.Errorf("placement %s already exists", placement.Name)
	}

	placementOrigin.Spec = placement.Spec
	_, err = clusterClient.ClusterV1beta1().Placements(o.Namespace).Update(context.TODO(), placementOrigin, metav1.UpdateOptions{})

	return err
}
