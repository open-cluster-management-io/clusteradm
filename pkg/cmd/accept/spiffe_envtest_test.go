// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func acceptOn(t *testing.T, cfg *rest.Config, trustDomain string, dryRun bool, cluster string) (string, error) {
	t.Helper()
	out := &bytes.Buffer{}
	o := NewOptions(envFlags(cfg), genericiooptions.IOStreams{Out: out, ErrOut: out})
	o.ClusteradmFlags.DryRun = dryRun
	o.SPIFFETrustDomain = trustDomain
	o.Values.Clusters = []string{cluster}
	err := o.Run(context.TODO())
	return out.String(), err
}

func createCluster(t *testing.T, client clusterclientset.Interface, name string, annotations map[string]string) {
	t.Helper()
	if _, err := client.ClusterV1().ManagedClusters().Create(context.TODO(), &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: annotations}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptSPIFFEAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	clusters := clusterclientset.NewForConfigOrDie(cfg)

	t.Run("an unpinned cluster is pinned and accepted, without a CSR", func(t *testing.T) {
		name := "spiffe-" + rand.String(5)
		createCluster(t, clusters, name, nil)
		out, err := acceptOn(t, cfg, "east.example.org", false, name)
		if err != nil {
			t.Fatal(err)
		}
		mc, _ := clusters.ClusterV1().ManagedClusters().Get(context.TODO(), name, metav1.GetOptions{})
		if mc.Annotations[spiffeTrustDomainAnnotation] != "east.example.org" || !mc.Spec.HubAcceptsClient {
			t.Errorf("cluster %+v", mc)
		}
		if !strings.Contains(out, "has joined the Hub") {
			t.Errorf("output:\n%s", out)
		}
	})

	t.Run("a cluster pinned to another trust domain is refused", func(t *testing.T) {
		name := "spiffe-" + rand.String(5)
		createCluster(t, clusters, name, map[string]string{spiffeTrustDomainAnnotation: "west.example.org"})
		if _, err := acceptOn(t, cfg, "east.example.org", false, name); err == nil || !strings.Contains(err.Error(), "west.example.org") {
			t.Errorf("error %v", err)
		}
	})

	t.Run("a dry run doesn't pin", func(t *testing.T) {
		name := "spiffe-" + rand.String(5)
		createCluster(t, clusters, name, nil)
		if _, err := acceptOn(t, cfg, "east.example.org", true, name); err != nil {
			t.Fatal(err)
		}
		mc, _ := clusters.ClusterV1().ManagedClusters().Get(context.TODO(), name, metav1.GetOptions{})
		if _, pinned := mc.Annotations[spiffeTrustDomainAnnotation]; pinned {
			t.Error("a dry run pinned the cluster")
		}
	})

	t.Run("a cluster that isn't spiffe needs its CSR approved", func(t *testing.T) {
		name := "csr-" + rand.String(5)
		createCluster(t, clusters, name, nil)
		if _, err := acceptOn(t, cfg, "", false, name); err == nil || !strings.Contains(err.Error(), "no csr is approved") {
			t.Errorf("error %v", err)
		}
	})

	t.Run("a hub that refuses the pin", func(t *testing.T) {
		name := "spiffe-refused-" + rand.String(5)
		createCluster(t, clusters, name, nil)
		denyPin(t, cfg, name)
		var err error
		// The policy takes effect after a moment.
		for i := 0; i < 30; i++ {
			if _, err = acceptOn(t, cfg, "east.example.org", false, name); err != nil {
				break
			}
			// The pin went through before the policy was active: start again.
			_ = clusters.ClusterV1().ManagedClusters().Delete(context.TODO(), name, metav1.DeleteOptions{})
			name = fmt.Sprintf("spiffe-refused-%s", rand.String(5))
			createCluster(t, clusters, name, nil)
			denyPin(t, cfg, name)
			time.Sleep(time.Second)
		}
		if err == nil || !strings.Contains(err.Error(), "fail to pin") {
			t.Errorf("error %v", err)
		}
	})
}

// denyPin installs a ValidatingAdmissionPolicy that refuses to pin cluster name.
func denyPin(t *testing.T, cfg *rest.Config, name string) {
	t.Helper()
	kube := kubernetes.NewForConfigOrDie(cfg)
	fail := admissionv1.Fail
	policy := &admissionv1.ValidatingAdmissionPolicy{ObjectMeta: metav1.ObjectMeta{Name: "deny-" + name},
		Spec: admissionv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: &fail,
			MatchConstraints: &admissionv1.MatchResources{ResourceRules: []admissionv1.NamedRuleWithOperations{{
				RuleWithOperations: admissionv1.RuleWithOperations{Operations: []admissionv1.OperationType{admissionv1.Update},
					Rule: admissionv1.Rule{APIGroups: []string{"cluster.open-cluster-management.io"}, APIVersions: []string{"v1"},
						Resources: []string{"managedclusters"}}}}}},
			Validations: []admissionv1.Validation{{
				Expression: fmt.Sprintf("object.metadata.name != '%s' || !has(object.metadata.annotations) || "+
					"!('%s' in object.metadata.annotations)", name, spiffeTrustDomainAnnotation),
				Message: "pinning is not allowed here"}},
		}}
	binding := &admissionv1.ValidatingAdmissionPolicyBinding{ObjectMeta: metav1.ObjectMeta{Name: "deny-" + name},
		Spec: admissionv1.ValidatingAdmissionPolicyBindingSpec{PolicyName: "deny-" + name,
			ValidationActions: []admissionv1.ValidationAction{admissionv1.Deny}}}
	if _, err := kube.AdmissionregistrationV1().ValidatingAdmissionPolicies().Create(context.TODO(), policy, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Create(context.TODO(), binding, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
}
