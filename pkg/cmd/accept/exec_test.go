// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"strings"
	"testing"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes/fake"

	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func TestApproveCSRPending(t *testing.T) {
	const clusterName = "cluster1"

	approvedCondition := certificatesv1.CertificateSigningRequestCondition{
		Type:   certificatesv1.CertificateApproved,
		Status: corev1.ConditionTrue,
	}
	deniedCondition := certificatesv1.CertificateSigningRequestCondition{
		Type:   certificatesv1.CertificateDenied,
		Status: corev1.ConditionTrue,
	}

	tests := []struct {
		name         string
		objects      func(t *testing.T) []runtime.Object
		dryRun       bool
		hubAccepts   bool
		wantApproved bool
		wantPending  bool
		wantJoin     bool
		wantErr      error
		wantContains []string
		wantExcludes []string
	}{
		{
			name:         "no csr on an already accepted cluster",
			objects:      func(*testing.T) []runtime.Object { return nil },
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  false,
			wantJoin:     true,
		},
		{
			name:         "no csr before the cluster is accepted",
			objects:      func(*testing.T) []runtime.Object { return nil },
			wantApproved: false,
			wantPending:  false,
			wantJoin:     false,
		},
		{
			name: "pending registration csr is approved",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{registrationCSR(t, "csr-pending", clusterName, "agent-a", nil)}
			},
			wantApproved: true,
			wantPending:  false,
			wantJoin:     true,
		},
		{
			name: "dry-run leaves a pending csr",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{registrationCSR(t, "csr-pending", clusterName, "agent-a", nil)}
			},
			dryRun:       true,
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  true,
			wantJoin:     false,
		},
		{
			name: "denied csr still blocks an already accepted cluster",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{registrationCSR(t, "csr-denied", clusterName, "agent-a", []certificatesv1.CertificateSigningRequestCondition{deniedCondition})}
			},
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  true,
			wantJoin:     false,
			wantErr:      errDeniedRegistrationCSR,
		},
		{
			name: "already approved csr",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{registrationCSR(t, "csr-approved", clusterName, "agent-a", []certificatesv1.CertificateSigningRequestCondition{approvedCondition})}
			},
			wantApproved: true,
			wantPending:  false,
			wantJoin:     true,
		},
		{
			name: "non-registration csr is ignored",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{&certificatesv1.CertificateSigningRequest{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "csr-other-user",
						Labels: map[string]string{clusterLabel: clusterName},
					},
					Spec: certificatesv1.CertificateSigningRequestSpec{
						Request:  certificateRequest(t, "admin"),
						Username: "system:admin",
						Groups:   []string{"system:masters"},
					},
				}}
			},
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  false,
			wantJoin:     true,
		},
		{
			name: "multiple requesters with a pending csr",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{
					registrationCSR(t, "csr-a", clusterName, "agent-a", []certificatesv1.CertificateSigningRequestCondition{approvedCondition}),
					registrationCSR(t, "csr-b", clusterName, "agent-b", nil),
				}
			},
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  true,
			wantJoin:     false,
			wantContains: []string{"approve is skipped"},
		},
		{
			name: "multiple requesters already approved and cluster accepted",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{
					registrationCSR(t, "csr-a", clusterName, "agent-a", []certificatesv1.CertificateSigningRequestCondition{approvedCondition}),
					registrationCSR(t, "csr-b", clusterName, "agent-b", []certificatesv1.CertificateSigningRequestCondition{approvedCondition}),
				}
			},
			hubAccepts:   true,
			wantApproved: true,
			wantPending:  false,
			wantJoin:     true,
			wantContains: []string{"CSR csr-a already approved", "CSR csr-b already approved"},
			wantExcludes: []string{"approve is skipped"},
		},
		{
			name: "csr for another cluster is ignored",
			objects: func(t *testing.T) []runtime.Object {
				return []runtime.Object{registrationCSR(t, "csr-other-cluster", "cluster2", "agent-a", nil)}
			},
			hubAccepts:   true,
			wantApproved: false,
			wantPending:  false,
			wantJoin:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			o := &Options{
				ClusteradmFlags: &genericclioptionsclusteradm.ClusteradmFlags{DryRun: tt.dryRun},
				Streams:         genericiooptions.IOStreams{Out: &out, ErrOut: &out},
			}
			kubeClient := fake.NewSimpleClientset(tt.objects(t)...)
			gotApproved, gotPending, err := o.approveCSR(context.Background(), kubeClient, clusterName, false)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("approveCSR() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("approveCSR() error = %v", err)
			}
			if gotApproved != tt.wantApproved || gotPending != tt.wantPending {
				t.Fatalf("approveCSR() = approved %v, pending %v, want approved %v, pending %v", gotApproved, gotPending, tt.wantApproved, tt.wantPending)
			}
			if gotJoin := gotApproved || (tt.hubAccepts && !gotPending); gotJoin != tt.wantJoin {
				t.Fatalf("joinApproved(%v, %v, %v) = %v, want %v", gotApproved, tt.hubAccepts, gotPending, gotJoin, tt.wantJoin)
			}
			for _, text := range tt.wantContains {
				if !strings.Contains(out.String(), text) {
					t.Fatalf("output %q does not contain %q", out.String(), text)
				}
			}
			for _, text := range tt.wantExcludes {
				if strings.Contains(out.String(), text) {
					t.Fatalf("output %q contains %q", out.String(), text)
				}
			}
		})
	}
}

func registrationCSR(t *testing.T, name, clusterName, commonName string, conditions []certificatesv1.CertificateSigningRequestCondition) *certificatesv1.CertificateSigningRequest {
	t.Helper()
	return &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{clusterLabel: clusterName},
		},
		Spec: certificatesv1.CertificateSigningRequestSpec{
			Request:  certificateRequest(t, commonName),
			Username: userNameSignatureBootstrapPrefix + commonName,
			Groups:   []string{groupNameBootstrap},
		},
		Status: certificatesv1.CertificateSigningRequestStatus{Conditions: conditions},
	}
}

func certificateRequest(t *testing.T, commonName string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: commonName},
	}, key)
	if err != nil {
		t.Fatalf("create certificate request: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
}
