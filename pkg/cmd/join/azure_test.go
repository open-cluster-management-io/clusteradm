// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

const (
	testPrincipalID = "11111111-1111-1111-1111-111111111111"
	testClientID    = "22222222-2222-2222-2222-222222222222"
	testTenantID    = "33333333-3333-3333-3333-333333333333"
)

func TestAzureOptionsValidate(t *testing.T) {
	cases := []struct {
		name      string
		opts      azureOptions
		expectErr string
	}{
		{
			name: "managed identity, system-assigned",
			opts: azureOptions{credential: "managed-identity-credential", principalID: testPrincipalID},
		},
		{
			name: "managed identity, user-assigned",
			opts: azureOptions{credential: "managed-identity-credential", principalID: testPrincipalID, clientID: testClientID},
		},
		{
			name: "workload identity",
			opts: azureOptions{credential: "workload-identity-credential", principalID: testPrincipalID, clientID: testClientID},
		},
		{
			name: "client secret",
			opts: azureOptions{credential: "environment-credential-secret", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID, clientSecretStdin: true},
		},
		{
			name: "client certificate with password and chain",
			opts: azureOptions{credential: "environment-credential-certificate", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID, clientCertPath: "cert.pem",
				clientCertPasswordStdin: true, clientSendCertChain: true},
		},
		{
			name:      "missing credential",
			opts:      azureOptions{principalID: testPrincipalID},
			expectErr: "--azure-credential is required",
		},
		{
			name:      "unknown credential",
			opts:      azureOptions{credential: "default-azure-credential", principalID: testPrincipalID},
			expectErr: "invalid --azure-credential",
		},
		{
			name:      "missing principal ID",
			opts:      azureOptions{credential: "managed-identity-credential"},
			expectErr: "--azure-principal-id is required",
		},
		{
			name:      "workload identity without client ID",
			opts:      azureOptions{credential: "workload-identity-credential", principalID: testPrincipalID},
			expectErr: "--azure-client-id is required",
		},
		{
			name: "client secret without tenant ID",
			opts: azureOptions{credential: "environment-credential-secret", principalID: testPrincipalID,
				clientID: testClientID, clientSecretStdin: true},
			expectErr: "--azure-client-id and --azure-tenant-id are required",
		},
		{
			name: "client secret without --azure-client-secret-stdin",
			opts: azureOptions{credential: "environment-credential-secret", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID},
			expectErr: "--azure-client-secret-stdin is required",
		},
		{
			name: "client certificate without its path",
			opts: azureOptions{credential: "environment-credential-certificate", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID},
			expectErr: "--azure-client-cert-path is required",
		},
		{
			name: "certificate flags with client secret",
			opts: azureOptions{credential: "environment-credential-secret", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID, clientSecretStdin: true, clientSendCertChain: true},
			expectErr: "certificate flags do not apply",
		},
		{
			name: "client secret flag with certificate",
			opts: azureOptions{credential: "environment-credential-certificate", principalID: testPrincipalID,
				clientID: testClientID, tenantID: testTenantID, clientCertPath: "cert.pem", clientSecretStdin: true},
			expectErr: "--azure-client-secret-stdin does not apply",
		},
		{
			name: "secret flags with workload identity",
			opts: azureOptions{credential: "workload-identity-credential", principalID: testPrincipalID,
				clientID: testClientID, clientCertPath: "cert.pem"},
			expectErr: "do not apply",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertErr(t, c.opts.validate(), c.expectErr)
		})
	}
}

func TestAzureOptionsUsed(t *testing.T) {
	if (&azureOptions{}).used() {
		t.Errorf("expected no azure flags to be reported as used")
	}
	if !(&azureOptions{clientSendCertChain: true}).used() {
		t.Errorf("expected --azure-client-send-cert-chain to be reported as used")
	}
}

func TestAzureRegistrationDriver(t *testing.T) {
	opts := azureOptions{credential: "environment-credential-secret", principalID: testPrincipalID,
		clientID: testClientID, tenantID: testTenantID, clientSecretStdin: true}
	expected := operatorv1.RegistrationDriver{
		AuthType: operatorv1.AzureAuthType,
		Azure: &operatorv1.AzureAuth{
			Credential:            operatorv1.AzureEnvironmentCredentialSecret,
			ManagedClusterAzureID: testPrincipalID,
			ClientID:              testClientID,
			TenantID:              testTenantID,
		},
	}
	if got := opts.registrationDriver(); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v, got %+v", expected, got)
	}
}

func TestAzureReadSecretMaterial(t *testing.T) {
	certPath := filepath.Join(t.TempDir(), "tls.crt")
	if err := os.WriteFile(certPath, []byte("certificate-and-key"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		opts       azureOptions
		stdin      string
		expectData map[string]string
		expectErr  string
	}{
		{
			name:       "client secret, trailing newline of a here-string dropped",
			opts:       azureOptions{credential: "environment-credential-secret", clientSecretStdin: true},
			stdin:      "s3cr3t\n",
			expectData: map[string]string{azureClientSecretKey: "s3cr3t"},
		},
		{
			name:       "client secret, inner whitespace kept",
			opts:       azureOptions{credential: "environment-credential-secret", clientSecretStdin: true},
			stdin:      " s3 cr3t \r\n",
			expectData: map[string]string{azureClientSecretKey: " s3 cr3t "},
		},
		{
			name:      "client secret, empty stdin",
			opts:      azureOptions{credential: "environment-credential-secret", clientSecretStdin: true},
			stdin:     "\n",
			expectErr: "nothing was read from stdin",
		},
		{
			name: "certificate without password",
			opts: azureOptions{credential: "environment-credential-certificate", clientCertPath: certPath},
			expectData: map[string]string{
				azureClientCertificateKey:          "certificate-and-key",
				azureClientSendCertificateChainKey: "false",
			},
		},
		{
			name: "certificate with password and chain",
			opts: azureOptions{credential: "environment-credential-certificate", clientCertPath: certPath,
				clientCertPasswordStdin: true, clientSendCertChain: true},
			stdin: "p4ss\n",
			expectData: map[string]string{
				azureClientCertificateKey:          "certificate-and-key",
				azureClientCertificatePasswordKey:  "p4ss",
				azureClientSendCertificateChainKey: "true",
			},
		},
		{
			name:      "certificate file missing",
			opts:      azureOptions{credential: "environment-credential-certificate", clientCertPath: certPath + ".missing"},
			expectErr: "failed to read --azure-client-cert-path",
		},
		{
			name: "workload identity needs no Secret",
			opts: azureOptions{credential: "workload-identity-credential"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.opts.readSecretMaterial(strings.NewReader(c.stdin))
			assertErr(t, err, c.expectErr)
			if c.expectErr != "" {
				return
			}
			got := map[string]string{}
			for k, v := range c.opts.secretData {
				got[k] = string(v)
			}
			if len(c.expectData) == 0 && len(got) == 0 {
				return
			}
			if !reflect.DeepEqual(got, c.expectData) {
				t.Errorf("expected secret data %v, got %v", c.expectData, got)
			}
		})
	}
}

func TestAzureApplyCredentialSecret(t *testing.T) {
	ctx := context.TODO()
	namespace := "open-cluster-management-agent"
	kubeClient := kubefake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: azureCredentialSecretName, Namespace: namespace},
		Data:       map[string][]byte{azureClientSecretKey: []byte("old")},
	})

	// An existing Secret, e.g. from an earlier join, is replaced with the new content.
	opts := azureOptions{secretData: map[string][]byte{azureClientSecretKey: []byte("new")}}
	if err := opts.applyCredentialSecret(ctx, kubeClient, namespace); err != nil {
		t.Fatalf("applyCredentialSecret failed: %v", err)
	}
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, azureCredentialSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(secret.Data[azureClientSecretKey]) != "new" {
		t.Errorf("expected the Secret to be updated, got %q", secret.Data[azureClientSecretKey])
	}

	// A credential without secret material creates nothing.
	emptyClient := kubefake.NewSimpleClientset()
	if err := (&azureOptions{}).applyCredentialSecret(ctx, emptyClient, namespace); err != nil {
		t.Fatalf("applyCredentialSecret failed: %v", err)
	}
	if secrets, _ := emptyClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{}); len(secrets.Items) != 0 {
		t.Errorf("expected no Secret, got %d", len(secrets.Items))
	}
}

func TestAgentNamespace(t *testing.T) {
	cases := []struct {
		mode      operatorv1.InstallMode
		name      string
		namespace string
		expected  string
	}{
		{operatorv1.InstallModeDefault, "klusterlet", "open-cluster-management-agent", "open-cluster-management-agent"},
		{operatorv1.InstallModeSingleton, "klusterlet", "", "open-cluster-management-agent"},
		{operatorv1.InstallModeHosted, "klusterlet-hosted-abc123", "open-cluster-management-klusterlet-hosted-abc123", "klusterlet-hosted-abc123"},
		{operatorv1.InstallModeSingletonHosted, "klusterlet-hosted-abc123", "", "klusterlet-hosted-abc123"},
	}
	for _, c := range cases {
		o := &Options{klusterletChartConfig: &chart.KlusterletChartConfig{
			Klusterlet: chart.KlusterletConfig{Mode: c.mode, Name: c.name, Namespace: c.namespace},
		}}
		if got := o.agentNamespace(); got != c.expected {
			t.Errorf("mode %s: expected agent namespace %q, got %q", c.mode, c.expected, got)
		}
	}
}

func assertErr(t *testing.T, err error, expect string) {
	t.Helper()
	switch {
	case expect == "" && err != nil:
		t.Fatalf("expected no error, got %v", err)
	case expect != "" && (err == nil || !strings.Contains(err.Error(), expect)):
		t.Fatalf("expected error containing %q, got %v", expect, err)
	}
}
