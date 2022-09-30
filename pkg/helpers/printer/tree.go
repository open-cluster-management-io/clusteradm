// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"context"
	"fmt"
	"strings"

	"github.com/disiqueira/gotree"
	"github.com/fatih/color"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
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

func PrintOperatorCRD(printer PrefixWriter, crdClient clientset.Interface, name string) error {
	crdList, err := crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	cmgr := operatorv1.RelatedResourceMeta{
		Resource: "customresourcedefinitions",
		Name:     name,
	}
	return printCRD(printer, crdList, []operatorv1.RelatedResourceMeta{cmgr})
}

func PrintComponentsCRD(printer PrefixWriter, crdClient clientset.Interface, resource []operatorv1.RelatedResourceMeta) error {
	crdList, err := crdClient.ApiextensionsV1().
		CustomResourceDefinitions().
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	return printCRD(printer, crdList, resource)
}

func printCRD(printer PrefixWriter, crdList *apiextensionsv1.CustomResourceDefinitionList, resource []operatorv1.RelatedResourceMeta) error {
	testingCRDNames := sets.NewString()
	for _, rs := range resource {
		if rs.Resource == "customresourcedefinitions" {
			testingCRDNames.Insert(rs.Name)
		}
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

func PrintComponentsDeploy(printer PrefixWriter, deployClient kubernetes.Interface, resource []operatorv1.RelatedResourceMeta, name string) error {
	var deploy operatorv1.RelatedResourceMeta
	for _, item := range resource {
		if item.Name == name {
			deploy = item
		}
	}

	client, err := deployClient.AppsV1().Deployments(deploy.Namespace).Get(context.TODO(), deploy.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	var pre string
	if strings.HasSuffix(name, "agent") {
		pre = "Agent:"
	} else if strings.HasSuffix(name, "controller") {
		pre = "Controller:"
	} else if strings.HasSuffix(name, "webhook") {
		pre = "Webhook:"
	}
	printer.Write(LEVEL_2, "%s\t(%d/%d) %s\n", pre, int(*client.Spec.Replicas), int(client.Status.AvailableReplicas), getImageName(client))

	return nil
}

func getImageName(deploy *appsv1.Deployment) string {
	imageName := "<none>"
	for _, container := range deploy.Spec.Template.Spec.Containers {
		imageName = container.Image
	}
	return imageName
}
