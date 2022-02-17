// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"fmt"

	"github.com/disiqueira/gotree"
	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	workapiv1 "open-cluster-management.io/api/work/v1"
)

func PrintWorkDetail(n gotree.Tree, work *workapiv1.ManifestWork) {
	condByRs := make(map[string][]workapiv1.ManifestCondition)
	for _, cond := range work.Status.ResourceStatus.Manifests {
		cond := cond
		groupResource := schema.GroupResource{Group: cond.ResourceMeta.Group, Resource: cond.ResourceMeta.Resource}.String()
		condByRs[groupResource] = append(condByRs[groupResource], cond)
	}
	for gr, rss := range condByRs {
		rsNode := n.Add(gr)
		for _, rs := range rss {
			identifier := rs.ResourceMeta.Name
			if len(rs.ResourceMeta.Namespace) > 0 {
				identifier = rs.ResourceMeta.Namespace + "/" + identifier
			}
			rsNode.Add(fmt.Sprintf("%s (%s)", identifier, getManifestResourceStatus(&rs)))
		}
	}
}

func getManifestResourceStatus(manifestCond *workapiv1.ManifestCondition) string {
	appliedCond := meta.FindStatusCondition(manifestCond.Conditions, workapiv1.WorkApplied)
	if appliedCond == nil {
		return "unknown"
	}
	if appliedCond.Status == metav1.ConditionTrue {
		return "applied"
	}
	return color.RedString("not-applied")
}
