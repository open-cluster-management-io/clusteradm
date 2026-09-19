// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"bytes"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	workapiv1 "open-cluster-management.io/api/work/v1"
)

func manifestCondition(namespace, name, resource string, applied *bool) workapiv1.ManifestCondition {
	mc := workapiv1.ManifestCondition{
		ResourceMeta: workapiv1.ManifestResourceMeta{
			Namespace: namespace,
			Name:      name,
			Resource:  resource,
		},
	}
	if applied != nil {
		status := metav1.ConditionFalse
		if *applied {
			status = metav1.ConditionTrue
		}
		mc.Conditions = []metav1.Condition{{Type: workapiv1.WorkApplied, Status: status}}
	}
	return mc
}

func TestWorkDetails(t *testing.T) {
	applied := true
	notApplied := false
	work := &workapiv1.ManifestWork{
		Status: workapiv1.ManifestWorkStatus{
			ResourceStatus: workapiv1.ManifestResourceStatus{
				Manifests: []workapiv1.ManifestCondition{
					manifestCondition("", "cm1", "configmaps", &applied),
					manifestCondition("ns1", "dep1", "deployments", &notApplied),
					manifestCondition("", "unknown1", "secrets", nil),
				},
			},
		},
	}

	details := WorkDetails(".manifests", work)

	if got := details[".manifests.configmaps.cm1"]; got != "applied" {
		t.Fatalf("cluster-scoped resource: got %v, want applied", got)
	}
	if got, ok := details[".manifests.deployments.ns1/dep1"]; !ok || !strings.Contains(got.(string), "not-applied") {
		t.Fatalf("namespaced resource: got %v, want it to contain not-applied", got)
	}
	if got := details[".manifests.secrets.unknown1"]; got != "unknown" {
		t.Fatalf("resource with no applied condition: got %v, want unknown", got)
	}
}

func TestFormatCRDVersion(t *testing.T) {
	servingVersions := map[string][]string{
		"widgets.example.com": {"v1", "v1beta1"},
	}
	storageVersion := map[string]string{
		"widgets.example.com": "v1",
	}

	got := formatCRDVersion(servingVersions, storageVersion, "widgets.example.com")

	if !strings.Contains(got, "*v1") {
		t.Fatalf("expected storage version to be marked with *, got %q", got)
	}
	if !strings.Contains(got, "v1beta1") {
		t.Fatalf("expected non-storage version to be listed unmarked, got %q", got)
	}
	if strings.Contains(got, "*v1beta1") {
		t.Fatalf("non-storage version should not be marked with *, got %q", got)
	}
}

func TestFormatCRDVersionUnknownName(t *testing.T) {
	got := formatCRDVersion(map[string][]string{}, map[string]string{}, "missing.example.com")
	if got != "" {
		t.Fatalf("expected empty string for an unknown CRD name, got %q", got)
	}
}

func TestGetImageName(t *testing.T) {
	cases := []struct {
		name   string
		deploy *appsv1.Deployment
		want   string
	}{
		{
			name: "no containers",
			deploy: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{}},
			},
			want: "<none>",
		},
		{
			name: "single container",
			deploy: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "registry/foo:v1"}}},
				}},
			},
			want: "registry/foo:v1",
		},
		{
			name: "last container wins",
			deploy: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{
						{Image: "registry/first:v1"},
						{Image: "registry/second:v1"},
					}},
				}},
			},
			want: "registry/second:v1",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getImageName(c.deploy); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestPrintCRD(t *testing.T) {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{
		Items: []apiextensionsv1.CustomResourceDefinition{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "installed.example.com"},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1", Served: true, Storage: true},
					},
				},
				Status: apiextensionsv1.CustomResourceDefinitionStatus{StoredVersions: []string{"v1"}},
			},
		},
	}
	resources := []operatorv1.RelatedResourceMeta{
		{Resource: "customresourcedefinitions", Name: "installed.example.com"},
		{Resource: "customresourcedefinitions", Name: "absent.example.com"},
	}

	buf := &bytes.Buffer{}
	if err := printCRD(NewPrefixWriter(buf), crdList, resources); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "(installed) installed.example.com") {
		t.Fatalf("expected installed CRD to be reported as installed, got:\n%s", out)
	}
	if !strings.Contains(out, "(absent) absent.example.com") {
		t.Fatalf("expected missing CRD to be reported as absent, got:\n%s", out)
	}
}
