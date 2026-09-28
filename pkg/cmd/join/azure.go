// Copyright Contributors to the Open Cluster Management project
package join

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	operatorv1 "open-cluster-management.io/api/operator/v1"
)

// The azure registration driver reads a service principal's secret material from this Secret in
// the agent namespace (see open-cluster-management.io/ocm/pkg/registration/register/azure_auth).
const (
	azureCredentialSecretName          = "azure-registration-credential"
	azureClientSecretKey               = "AZURE_CLIENT_SECRET"
	azureClientCertificateKey          = "tls.crt"
	azureClientCertificatePasswordKey  = "AZURE_CLIENT_CERTIFICATE_PASSWORD"
	azureClientSendCertificateChainKey = "AZURE_CLIENT_SEND_CERTIFICATE_CHAIN"
)

// azureOptions holds the join flags of the azure registration driver.
type azureOptions struct {
	credential              string
	principalID             string
	clientID                string
	tenantID                string
	clientSecretStdin       bool
	clientCertPath          string
	clientCertPasswordStdin bool
	clientSendCertChain     bool

	// secretData is the content of the credential Secret, read by readSecretMaterial. It never
	// goes into the Klusterlet or the rendered manifests.
	secretData map[string][]byte
}

func (a *azureOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.credential, "azure-credential", "", fmt.Sprintf(
		"The Azure credential the agents authenticate with when --registration-auth is azure: %s, %s, %s or %s.",
		operatorv1.AzureManagedIdentityCredential, operatorv1.AzureWorkloadIdentityCredential,
		operatorv1.AzureEnvironmentCredentialSecret, operatorv1.AzureEnvironmentCredentialCertificate))
	fs.StringVar(&a.principalID, "azure-principal-id", "",
		"The Azure AD object (principal) ID of the identity; the hub grants its permissions to it. Required when --registration-auth is azure.")
	fs.StringVar(&a.clientID, "azure-client-id", "",
		"The client ID of the identity. Required unless --azure-credential is managed-identity-credential, where omitting it selects the system-assigned identity.")
	fs.StringVar(&a.tenantID, "azure-tenant-id", "",
		"The Entra ID tenant ID. Required for environment-credential-secret and environment-credential-certificate.")
	fs.BoolVar(&a.clientSecretStdin, "azure-client-secret-stdin", false,
		"Read the service principal client secret from stdin. Required for environment-credential-secret.")
	fs.StringVar(&a.clientCertPath, "azure-client-cert-path", "",
		"Path to the service principal certificate and its private key, PEM or PKCS#12. Required for environment-credential-certificate.")
	fs.BoolVar(&a.clientCertPasswordStdin, "azure-client-cert-password-stdin", false,
		"Read the password of a PKCS#12 certificate from stdin. Only for environment-credential-certificate.")
	fs.BoolVar(&a.clientSendCertChain, "azure-client-send-cert-chain", false,
		"Send the certificate chain with each token request. Only for environment-credential-certificate.")
}

// used reports whether any azure flag is set.
func (a *azureOptions) used() bool {
	return a.credential != "" || a.principalID != "" || a.clientID != "" || a.tenantID != "" ||
		a.clientSecretStdin || a.clientCertPath != "" || a.clientCertPasswordStdin || a.clientSendCertChain
}

// validate checks that the flags required by the selected credential are set and that no flag of
// another credential is.
func (a *azureOptions) validate() error {
	if a.principalID == "" {
		return fmt.Errorf("--azure-principal-id is required when registration-auth is azure")
	}

	credential := operatorv1.AzureCredentialType(a.credential)
	secretFlag := a.clientSecretStdin
	certFlags := a.clientCertPath != "" || a.clientCertPasswordStdin || a.clientSendCertChain
	switch credential {
	case operatorv1.AzureManagedIdentityCredential, operatorv1.AzureWorkloadIdentityCredential:
		if credential == operatorv1.AzureWorkloadIdentityCredential && a.clientID == "" {
			return fmt.Errorf("--azure-client-id is required for --azure-credential %s", credential)
		}
		if secretFlag || certFlags {
			return fmt.Errorf("the client secret and certificate flags do not apply to --azure-credential %s", credential)
		}
	case operatorv1.AzureEnvironmentCredentialSecret:
		if a.clientID == "" || a.tenantID == "" {
			return fmt.Errorf("--azure-client-id and --azure-tenant-id are required for --azure-credential %s", credential)
		}
		if !secretFlag {
			return fmt.Errorf("--azure-client-secret-stdin is required for --azure-credential %s", credential)
		}
		if certFlags {
			return fmt.Errorf("the certificate flags do not apply to --azure-credential %s", credential)
		}
	case operatorv1.AzureEnvironmentCredentialCertificate:
		if a.clientID == "" || a.tenantID == "" {
			return fmt.Errorf("--azure-client-id and --azure-tenant-id are required for --azure-credential %s", credential)
		}
		if a.clientCertPath == "" {
			return fmt.Errorf("--azure-client-cert-path is required for --azure-credential %s", credential)
		}
		if secretFlag {
			return fmt.Errorf("--azure-client-secret-stdin does not apply to --azure-credential %s", credential)
		}
	case "":
		return fmt.Errorf("--azure-credential is required when registration-auth is azure")
	default:
		return fmt.Errorf("invalid --azure-credential %q", a.credential)
	}
	return nil
}

// registrationDriver returns the Klusterlet registration driver. It carries only the identifying
// values; secret material goes into the credential Secret.
func (a *azureOptions) registrationDriver() operatorv1.RegistrationDriver {
	return operatorv1.RegistrationDriver{
		AuthType: operatorv1.AzureAuthType,
		Azure: &operatorv1.AzureAuth{
			Credential:            operatorv1.AzureCredentialType(a.credential),
			ManagedClusterAzureID: a.principalID,
			ClientID:              a.clientID,
			TenantID:              a.tenantID,
		},
	}
}

// readSecretMaterial reads the client secret or certificate password from stdin and the
// certificate from its file into secretData. Secrets are only read from stdin, never from flag
// values, so they do not appear in the process arguments or the shell history.
func (a *azureOptions) readSecretMaterial(stdin io.Reader) error {
	switch operatorv1.AzureCredentialType(a.credential) {
	case operatorv1.AzureEnvironmentCredentialSecret:
		secret, err := readStdinSecret(stdin, "--azure-client-secret-stdin")
		if err != nil {
			return err
		}
		a.secretData = map[string][]byte{azureClientSecretKey: secret}
	case operatorv1.AzureEnvironmentCredentialCertificate:
		cert, err := os.ReadFile(a.clientCertPath)
		if err != nil {
			return fmt.Errorf("failed to read --azure-client-cert-path %q: %w", a.clientCertPath, err)
		}
		if len(cert) == 0 {
			return fmt.Errorf("--azure-client-cert-path %q is empty", a.clientCertPath)
		}
		a.secretData = map[string][]byte{
			azureClientCertificateKey:          cert,
			azureClientSendCertificateChainKey: []byte(strconv.FormatBool(a.clientSendCertChain)),
		}
		if a.clientCertPasswordStdin {
			password, err := readStdinSecret(stdin, "--azure-client-cert-password-stdin")
			if err != nil {
				return err
			}
			a.secretData[azureClientCertificatePasswordKey] = password
		}
	}
	return nil
}

// readStdinSecret reads a secret value from stdin, dropping the trailing newline a here-string or
// echo adds.
func readStdinSecret(stdin io.Reader, flag string) ([]byte, error) {
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s from stdin: %w", flag, err)
	}
	value := strings.TrimRight(string(data), "\r\n")
	if value == "" {
		return nil, fmt.Errorf("%s is set but nothing was read from stdin", flag)
	}
	return []byte(value), nil
}

// applyCredentialSecret creates or updates the credential Secret in the agent namespace, when the
// selected credential needs one.
func (a *azureOptions) applyCredentialSecret(ctx context.Context, kubeClient kubernetes.Interface, namespace string) error {
	if len(a.secretData) == 0 {
		return nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: azureCredentialSecretName, Namespace: namespace},
		Type:       corev1.SecretTypeOpaque,
		Data:       a.secretData,
	}
	_, err := kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		existing, getErr := kubeClient.CoreV1().Secrets(namespace).Get(ctx, azureCredentialSecretName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}
		existing.Data = a.secretData
		_, err = kubeClient.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	}
	if err != nil {
		return fmt.Errorf("failed to apply Secret %s/%s: %w", namespace, azureCredentialSecretName, err)
	}
	return nil
}
