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
