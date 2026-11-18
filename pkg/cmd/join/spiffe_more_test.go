// Copyright Contributors to the Open Cluster Management project
package join

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	fakeoperator "open-cluster-management.io/api/client/operator/clientset/versioned/fake"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func TestSPIFFEJoinFlagErrors(t *testing.T) {
	cases := []struct {
		name   string
		modify func(o *Options)
		want   string
	}{
		{"bad trust domain", func(o *Options) { o.spiffe.trustDomain = "Not A Domain" }, "--spiffe-trust-domain"},
		{"bad add-on template", func(o *Options) {
			o.spiffe.addOnAgentIDTemplate = "spiffe://{trustDomain}/ocm/{cluster}/{addon}"
		}, "--spiffe-addon-agent-id-template"},
		{"bad admission enforcement", func(o *Options) { o.spiffe.admissionEnforcement = "None" }, "--spiffe-admission-enforcement"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, cmd, _ := spiffeJoinOptions(t)
			c.modify(o)
			if err := o.complete(cmd, nil); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want %q", err, c.want)
			}
		})
	}
}

// A feature gate already listed (for example disabled) is switched on, not added twice.
func TestEnableFeatureExisting(t *testing.T) {
	gates := enableFeature([]operatorv1.FeatureGate{{Feature: "SPIFFERegistration", Mode: operatorv1.FeatureGateModeTypeDisable}},
		"SPIFFERegistration")
	if len(gates) != 1 || gates[0].Mode != operatorv1.FeatureGateModeTypeEnable {
		t.Errorf("gates %+v", gates)
	}
}

func TestWaitForSPIFFEAdmission(t *testing.T) {
	k := &operatorv1.Klusterlet{ObjectMeta: metav1.ObjectMeta{Name: "klusterlet"}}
	client := fakeoperator.NewSimpleClientset(k)
	gets := 0
	client.PrependReactor("get", "klusterlets", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		got := k.DeepCopy()
		reason, status := "Recovering", metav1.ConditionFalse
		if gets > 1 {
			reason, status = "Recorded", metav1.ConditionTrue
		}
		meta.SetStatusCondition(&got.Status.Conditions, metav1.Condition{
			Type: operatorv1.ConditionSPIFFEAdmissionReady, Status: status, Reason: reason})
		return true, got, nil
	})
	out := &bytes.Buffer{}
	got, err := waitForSPIFFEAdmission(context.Background(), out, client, 10, "klusterlet")
	if err != nil || got == nil || gets < 2 {
		t.Fatalf("klusterlet %v, err %v after %d reads", got, err, gets)
	}

	// Never ready: a timeout that says what it waited for.
	notReady := fakeoperator.NewSimpleClientset(k)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := waitForSPIFFEAdmission(ctx, out, notReady, 1, "klusterlet"); err == nil {
		t.Error("no error although the Klusterlet never became ready")
	}
}

func TestPhaseStatus(t *testing.T) {
	phase := &atomic.Value{}
	phase.Store("Recovering")
	if got := phaseStatus(phase)(); got != "Recovering" {
		t.Errorf("status %q", got)
	}
}

// A failed read while waiting is retried.
func TestWaitForSPIFFEAdmissionReadError(t *testing.T) {
	k := &operatorv1.Klusterlet{ObjectMeta: metav1.ObjectMeta{Name: "klusterlet"}}
	meta.SetStatusCondition(&k.Status.Conditions, metav1.Condition{
		Type: operatorv1.ConditionSPIFFEAdmissionReady, Status: metav1.ConditionTrue, Reason: "Recorded"})
	client := fakeoperator.NewSimpleClientset(k)
	failed := false
	client.PrependReactor("get", "klusterlets", func(clienttesting.Action) (bool, runtime.Object, error) {
		if !failed {
			failed = true
			return true, nil, errors.New("connection refused")
		}
		return false, nil, nil
	})
	if got, err := waitForSPIFFEAdmission(context.Background(), &bytes.Buffer{}, client, 10, "klusterlet"); err != nil || got == nil {
		t.Errorf("klusterlet %v, err %v", got, err)
	}
}

// --spiffe-admission-enforcement reaches the Klusterlet, and is required.
func TestSPIFFEJoinAdmissionEnforcement(t *testing.T) {
	for _, mode := range []string{"ValidatingAdmissionPolicy", "ValidatingWebhook"} {
		o, cmd, _ := spiffeJoinOptions(t)
		o.spiffe.admissionEnforcement = mode
		if err := o.complete(cmd, nil); err != nil {
			t.Fatal(err)
		}
		got := o.klusterletChartConfig.Klusterlet.RegistrationConfiguration.RegistrationDriver.SPIFFE.AdmissionEnforcement
		if string(got) != mode {
			t.Errorf("admissionEnforcement %q, want %q", got, mode)
		}
	}
	o, cmd, _ := spiffeJoinOptions(t)
	o.spiffe.admissionEnforcement = ""
	if err := o.complete(cmd, nil); err == nil || !strings.Contains(err.Error(), "--spiffe-admission-enforcement is required") {
		t.Errorf("error %v, want the flag required", err)
	}
}

// ValidatingAdmissionPolicy enforcement on a managed cluster older than 1.30 is refused before
// anything is applied, naming the version.
func TestCheckSPIFFEEnforcement(t *testing.T) {
	o, _, _ := spiffeJoinOptions(t)
	served := fakekube.NewSimpleClientset()
	served.Resources = []*metav1.APIResourceList{{GroupVersion: "admissionregistration.k8s.io/v1",
		APIResources: []metav1.APIResource{{Name: "validatingwebhookconfigurations"}, {Name: "validatingadmissionpolicies"}}}}
	if err := o.checkSPIFFEEnforcement(clientOf(served)); err != nil {
		t.Errorf("1.30 or later: %v", err)
	}

	old := fakekube.NewSimpleClientset()
	old.Resources = []*metav1.APIResourceList{{GroupVersion: "admissionregistration.k8s.io/v1",
		APIResources: []metav1.APIResource{{Name: "validatingwebhookconfigurations"}}}}
	old.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "v1.29.14"}
	err := o.checkSPIFFEEnforcement(clientOf(old))
	if err == nil || !strings.HasPrefix(err.Error(), "unsupported Kubernetes version v1.29.14") ||
		!strings.Contains(err.Error(), "use ValidatingWebhook") {
		t.Errorf("1.29: %v", err)
	}

	// No admissionregistration.k8s.io/v1 at all, and no version.
	none := fakekube.NewSimpleClientset()
	none.PrependReactor("get", "version", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("injected")
	})
	if err := o.checkSPIFFEEnforcement(clientOf(none)); err == nil || !strings.HasPrefix(err.Error(), "unsupported Kubernetes version unknown") {
		t.Errorf("no API group: %v", err)
	}

	broken := fakekube.NewSimpleClientset()
	broken.PrependReactor("get", "resource", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("injected")
	})
	if err := o.checkSPIFFEEnforcement(clientOf(broken)); err == nil || !strings.Contains(err.Error(), "failed to check") {
		t.Errorf("discovery error: %v", err)
	}

	if err := o.checkSPIFFEEnforcement(func() (kubernetes.Interface, error) { return nil, errors.New("no kubeconfig") }); err == nil {
		t.Error("no error without a client")
	}

	o.spiffe.admissionEnforcement = "ValidatingWebhook"
	if err := o.checkSPIFFEEnforcement(func() (kubernetes.Interface, error) { panic("the webhook needs no client") }); err != nil {
		t.Errorf("the webhook on 1.29: %v", err)
	}
}

func clientOf(c kubernetes.Interface) func() (kubernetes.Interface, error) {
	return func() (kubernetes.Interface, error) { return c, nil }
}

// While the operator's webhook waits for its SVID, join says once which SPIRE entry it needs,
// and goes on waiting.
func TestWaitForSPIFFEAdmissionWebhookNotReady(t *testing.T) {
	k := &operatorv1.Klusterlet{ObjectMeta: metav1.ObjectMeta{Name: "klusterlet"}}
	client := fakeoperator.NewSimpleClientset(k)
	gets := 0
	client.PrependReactor("get", "klusterlets", func(clienttesting.Action) (bool, runtime.Object, error) {
		gets++
		got := k.DeepCopy()
		status, reason := metav1.ConditionFalse, reasonWebhookNotReady
		if gets > 2 {
			status, reason = metav1.ConditionTrue, "Recorded"
		}
		meta.SetStatusCondition(&got.Status.Conditions, metav1.Condition{Type: operatorv1.ConditionSPIFFEAdmissionReady,
			Status: status, Reason: reason, Message: "create a SPIRE registration entry for ServiceAccount klusterlet"})
		return true, got, nil
	})
	out := &bytes.Buffer{}
	got, err := waitForSPIFFEAdmission(context.Background(), out, client, 20, "klusterlet")
	if err != nil || got == nil {
		t.Fatalf("klusterlet %v, err %v", got, err)
	}
	if n := strings.Count(out.String(), "waits for its operator's webhook"); n != 1 {
		t.Errorf("said it %d times:\n%s", n, out)
	}
}

// While the operator refuses the Klusterlet, join's timeout says why.
func TestWaitForSPIFFEAdmissionRefused(t *testing.T) {
	k := &operatorv1.Klusterlet{ObjectMeta: metav1.ObjectMeta{Name: "klusterlet"}}
	meta.SetStatusCondition(&k.Status.Conditions, metav1.Condition{Type: operatorv1.ConditionKlusterletApplied,
		Status: metav1.ConditionFalse, Reason: "SPIFFEPermissionsMissing", Message: "bind klusterlet-spiffe"})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := waitForSPIFFEAdmission(ctx, &bytes.Buffer{}, fakeoperator.NewSimpleClientset(k), 1, "klusterlet")
	if err == nil || !strings.Contains(err.Error(), "SPIFFEPermissionsMissing: bind klusterlet-spiffe") {
		t.Errorf("err %v, want the refusal's reason and message", err)
	}
}

func TestSPIFFENextStepsWithWebhook(t *testing.T) {
	o, _, _ := spiffeJoinOptions(t)
	k := &operatorv1.Klusterlet{Status: operatorv1.KlusterletStatus{SPIFFE: &operatorv1.KlusterletSPIFFEStatus{
		AdmissionEnforcement: operatorv1.SPIFFEAdmissionValidatingWebhook,
		WebhookSPIFFEID:      "spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/operator",
		WebhookDNSName:       "klusterlet-spiffe-webhook.open-cluster-management.svc"}}}
	out := &bytes.Buffer{}
	o.printSPIFFENextSteps(out, k)
	if !strings.Contains(out.String(), "serves as spiffe://ocm-hub.example.org/ocm/cluster/cluster1/agent/operator with DNS name "+
		"klusterlet-spiffe-webhook.open-cluster-management.svc") {
		t.Errorf("output:\n%s", out)
	}
	out.Reset()
	o.printSPIFFENextSteps(out, &operatorv1.Klusterlet{})
	if strings.Contains(out.String(), "webhook") {
		t.Errorf("the webhook mentioned for a Klusterlet without it:\n%s", out)
	}
}

// join's validation runs the version check: with ValidatingAdmissionPolicy it asks the managed
// cluster, here an address nothing listens on, so it fails before anything is applied.
func TestSPIFFEJoinValidateChecksEnforcement(t *testing.T) {
	o, cmd, _ := spiffeJoinOptions(t)
	if err := o.complete(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "failed to check the managed cluster's admission APIs") {
		t.Errorf("validate: %v", err)
	}
}
