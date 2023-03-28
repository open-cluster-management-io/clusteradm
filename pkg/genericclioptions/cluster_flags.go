// Copyright Contributors to the Open Cluster Management project
package genericclioptions

import (
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"
)

type ClusterOption struct {
	Cluster    string
	Clusters   []string
	allowUnset bool
}

func NewClusterOption() *ClusterOption {
	return &ClusterOption{}
}

func (c *ClusterOption) AllowUnset() *ClusterOption {
	c.allowUnset = true
	return c
}

func (c *ClusterOption) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.Cluster, "cluster", "c", "", "Name of the managed cluster")
	flags.StringSliceVar(&c.Clusters, "clusters", []string{}, "A list of the managed clusters.")
}

func (c *ClusterOption) AllClusters() sets.String {
	output := sets.NewString(c.Clusters...)
	if len(c.Cluster) != 0 {
		output.Insert(c.Cluster)
	}

	return output
}

func (c *ClusterOption) Validate() error {
	for _, cluster := range c.Clusters {
		if len(cluster) == 0 {
			return fmt.Errorf("--clusters cannot be set as an empty value")
		}
	}
	if len(c.Cluster) == 0 && len(c.Clusters) == 0 && !c.allowUnset {
		return fmt.Errorf("either --cluster or --clusters needs to be set")
	}

	return nil
}
