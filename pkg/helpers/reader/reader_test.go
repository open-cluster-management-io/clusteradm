// Copyright Contributors to the Open Cluster Management project

package reader

import (
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	kubectlutil "k8s.io/kubectl/pkg/util"
)

// last-applied-configuration as written by `kubectl apply` of a live capture
// (`kubectl get klusterlet klusterlet -o yaml > klusterlet.yaml`).
const capturedObject = `{"apiVersion":"operator.open-cluster-management.io/v1","kind":"Klusterlet",` +
	`"metadata":{"creationTimestamp":"2026-10-08T16:11:33Z",` +
	`"finalizers":["operator.open-cluster-management.io/klusterlet-cleanup"],"name":"klusterlet",` +
	`"resourceVersion":"5736","uid":"30a9c771-0e70-41b9-b50d-c1921543e59c"},` +
	`"spec":{"clusterName":"spoke"},"status":{"conditions":[{"type":"Applied"}]}}`

// what the klusterlet chart renders, minus the annotation
const renderedManifest = `{"apiVersion":"operator.open-cluster-management.io/v1","kind":"Klusterlet",` +
	`"metadata":{"name":"klusterlet"},"spec":{"clusterName":"spoke"}}`

func klusterletWithAnnotation(annotation string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "operator.open-cluster-management.io/v1",
		"kind":       "Klusterlet",
		"metadata": map[string]interface{}{
			"name":            "klusterlet",
			"resourceVersion": "5736",
			"uid":             "30a9c771-0e70-41b9-b50d-c1921543e59c",
			"finalizers":      []interface{}{"operator.open-cluster-management.io/klusterlet-cleanup"},
			"annotations":     map[string]interface{}{corev1.LastAppliedConfigAnnotation: annotation},
		},
		"spec": map[string]interface{}{"clusterName": "spoke"},
	}}
}

func lastApplied(obj *unstructured.Unstructured) string {
	annots, err := meta.NewAccessor().Annotations(obj)
	if err != nil {
		panic(err)
	}
	return annots[corev1.LastAppliedConfigAnnotation]
}

func TestDropServerOwnedMetadataFromLastApplied(t *testing.T) {
	tests := []struct {
		name        string
		annotation  string
		wantChanged bool
	}{
		{"absent annotation", `{}`, false},
		{"empty annotation", ``, false},
		{"malformed annotation", `{not json`, false},
		{"only name", `{"metadata":{"name":"klusterlet"}}`, false},
		{"unprunable metadata dropped", capturedObject, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := klusterletWithAnnotation(tt.annotation)
			dropUnprunableMetadataFromLastApplied(obj)

			got := lastApplied(obj)
			if !tt.wantChanged {
				if got != tt.annotation {
					t.Fatalf("annotation changed to %q, want it left alone", got)
				}
				return
			}

			var applied map[string]interface{}
			if err := json.Unmarshal([]byte(got), &applied); err != nil {
				t.Fatalf("annotation is not valid json after sanitizing: %v", err)
			}
			metadata := applied["metadata"].(map[string]interface{})
			for _, key := range unprunableMetadata {
				if _, present := metadata[key]; present {
					t.Errorf("%q survived sanitizing: %s", key, got)
				}
			}
			// the fields the annotation legitimately describes are kept, including the
			// server-populated ones the API server ignores when a patch deletes them
			if metadata["name"] != "klusterlet" {
				t.Errorf("name was dropped: %v", metadata)
			}
			if metadata["creationTimestamp"] != "2026-10-08T16:11:33Z" {
				t.Errorf("creationTimestamp was dropped: %v", metadata)
			}
			if applied["spec"] == nil {
				t.Errorf("spec was dropped: %s", got)
			}
		})
	}
}

// The patch sent to the API server must not ask it to delete the fields it or another
// controller owns - a null resourceVersion is rejected with "metadata.resourceVersion:
// Invalid value: 0: must be specified for an update", and a null finalizers list is
// honored, which would drop the Klusterlet CR's cleanup finalizer. Everything else the
// three-way merge deletes has to stay.
func TestPatchHasNoUnprunableMetadata(t *testing.T) {
	obj := klusterletWithAnnotation(capturedObject)
	current, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		t.Fatalf("encoding current object: %v", err)
	}

	patch := func() string {
		original, err := kubectlutil.GetOriginalConfiguration(obj)
		if err != nil {
			t.Fatalf("reading original configuration: %v", err)
		}
		patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(
			original, []byte(renderedManifest), current)
		if err != nil {
			t.Fatalf("computing patch: %v", err)
		}
		return string(patch)
	}

	before := patch()
	for _, want := range []string{`"resourceVersion":null`, `"finalizers":null`} {
		if !strings.Contains(before, want) {
			t.Fatalf("test setup is wrong, pre-fix patch has no %s: %s", want, before)
		}
	}

	dropUnprunableMetadataFromLastApplied(obj)

	after := patch()
	for _, key := range unprunableMetadata {
		if strings.Contains(after, `"`+key+`"`) {
			t.Errorf("patch still asks to delete %s: %s", key, after)
		}
	}
	if !strings.Contains(after, `"status":null`) {
		t.Errorf("sanitizing removed the rest of the deletion patch: %s", after)
	}
}
