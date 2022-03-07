// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/disiqueira/gotree"
	"github.com/fatih/color"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	operatorv1 "open-cluster-management.io/api/operator/v1"
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

func PrintComponentsCRD(printer PrefixWriter, crdClient clientset.Interface, resource []operatorv1.RelatedResourceMeta) error {
	testingCRDNames := sets.NewString()
	for _, rs := range resource {
		if rs.Resource == "customresourcedefinitions" {
			testingCRDNames.Insert(rs.Name)
		}
	}
	crdList, err := crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	statuses := make(map[string]string)
	existingCRDNames := sets.NewString()
	crdVersions := make(map[string][]string)
	crdStorageVersion := make(map[string]string)
	for _, existingCRD := range crdList.Items {
		existingCRDNames.Insert(existingCRD.Name)
		crdVersions[existingCRD.Name] = existingCRD.Status.StoredVersions
		servingVersions := sets.NewString()
		for _, v := range existingCRD.Spec.Versions {
			if v.Served {
				servingVersions.Insert(v.Name)
			}
			if v.Storage {
				crdStorageVersion[existingCRD.Name] = v.Name
			}
			crdVersions[existingCRD.Name] = servingVersions.List()
		}
	}
	for _, name := range testingCRDNames.List() {
		st := "absent"
		if existingCRDNames.Has(name) {
			st = "installed"
		}
		statuses[name] = st
	}
	printer.Write(LEVEL_1, "CustomResourceDefinition:\n")
	for name, st := range statuses {
		versionStr := formatCRDVersion(crdVersions, crdStorageVersion, name)
		printer.Write(LEVEL_2, "(%s) %s [%s]\n", st, name, versionStr)
	}
	return nil
}

func formatCRDVersion(allServingVersions map[string][]string, storageVersion map[string]string, crdName string) string {
	servings := allServingVersions[crdName]
	storage := storageVersion[crdName]
	outputVersions := sets.NewString()
	for _, v := range servings {
		if v == storage {
			v = "*" + v
		}
		outputVersions.Insert(v)
	}
	return strings.Join(outputVersions.List(), "|")
}
