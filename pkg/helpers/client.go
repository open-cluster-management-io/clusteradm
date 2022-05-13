// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	apiextensionshelpers "k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/config"
)

type TokenType string

const (
	BootstrapToken      TokenType = "bootstrap-token"
	ServiceAccountToken TokenType = "service-account-token"
	UnknownToken        TokenType = "unknown-token"
)

func GetClients(f util.Factory) (
	kubeClient kubernetes.Interface,
	apiExtensionsClient apiextensionsclient.Interface,
	dynamicClient dynamic.Interface,
	err error) {
	kubeClient, err = f.KubernetesClientSet()
	if err != nil {
		return
	}
	dynamicClient, err = f.DynamicClient()
	if err != nil {
		return
	}

	var restConfig *rest.Config
	restConfig, err = f.ToRESTConfig()
	if err != nil {
		return
	}

	apiExtensionsClient, err = apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return
	}
	return
}

//GetAPIServer gets the api server url
func GetAPIServer(kubeClient kubernetes.Interface) (string, error) {
	config, err := getClusterInfoKubeConfig(kubeClient)
	if err != nil {
		return "", err
	}
	clusters := config.Clusters
	if len(clusters) != 1 {
		return "", fmt.Errorf("can not find the cluster in the cluster-info")
	}
	cluster := clusters[0].Cluster
	return cluster.Server, nil
}

//GetCACert returns the CA cert.
//First by looking in the cluster-info configmap of the kube-public ns and if not found,
//it searches in the kube-root-ca.crt configmap.
func GetCACert(kubeClient kubernetes.Interface) ([]byte, error) {
	config, err := getClusterInfoKubeConfig(kubeClient)
	if err == nil {
		clusters := config.Clusters
		if len(clusters) != 1 {
			return nil, fmt.Errorf("can not find the cluster in the cluster-info")
		}
		cluster := clusters[0].Cluster
		return cluster.CertificateAuthorityData, nil
	}
	if errors.IsNotFound(err) {
		cm, err := kubeClient.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "kube-root-ca.crt", metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return []byte(cm.Data["ca.crt"]), nil
	}
	return nil, err
}

func getClusterInfoKubeConfig(kubeClient kubernetes.Interface) (*clientcmdapiv1.Config, error) {
	cm, err := kubeClient.CoreV1().ConfigMaps("kube-public").Get(context.TODO(), "cluster-info", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	config := &clientcmdapiv1.Config{}
	err = yaml.Unmarshal([]byte(cm.Data["kubeconfig"]), config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

//WaitCRDToBeReady waits if a crd is ready
func WaitCRDToBeReady(apiExtensionsClient apiextensionsclient.Interface, name string, b wait.Backoff) error {
	errGet := retry.OnError(b, func(err error) bool {
		if err != nil {
			fmt.Printf("Wait  for %s crd to be ready\n", name)
			return true
		}
		return false
	}, func() error {
		crd, err := apiExtensionsClient.ApiextensionsV1().CustomResourceDefinitions().
			Get(context.TODO(),
				name,
				metav1.GetOptions{})
		if established := apiextensionshelpers.IsCRDConditionTrue(crd, apiextensionsv1.Established); !established {
			fmt.Printf("Wait  for %s crd to be established\n", name)
			return fmt.Errorf("Wait  for %s crd to be established", name)
		}

		return err
	})
	return errGet
}

//GetToken returns the bootstrap token.
//It searchs first for the service-account token and then if it is not found
//it looks for the bootstrap token in kube-system.
func GetToken(kubeClient kubernetes.Interface) (string, TokenType, error) {
	token, err := GetBootstrapTokenFromSA(kubeClient)
	if err != nil {
		if errors.IsNotFound(err) {
			//As no SA search for bootstrap token
			var token string
			token, err = GetBootstrapToken(kubeClient)
			if err == nil {
				return token, BootstrapToken, nil
			}
		}
		return "", UnknownToken, err
	}
	return token, ServiceAccountToken, nil
}

//GetBootstrapSecret returns the secret in kube-system
func GetBootstrapSecret(kubeClient kubernetes.Interface) (*corev1.Secret, error) {
	var bootstrapSecret *corev1.Secret
	l, err := kubeClient.CoreV1().
		Secrets("kube-system").
		List(context.TODO(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%v = %v", config.LabelApp, config.ClusterManagerName)})
	if err != nil {
		return nil, err
	}
	for _, s := range l.Items {
		if strings.HasPrefix(s.Name, config.BootstrapSecretPrefix) {
			bootstrapSecret = &s
		}
	}
	if bootstrapSecret == nil {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    corev1.GroupName,
			Resource: "secrets"},
			fmt.Sprintf("%s*", config.BootstrapSecretPrefix))

	}
	return bootstrapSecret, err
}

//GetBootstrapToken returns the token in kube-system
func GetBootstrapToken(kubeClient kubernetes.Interface) (string, error) {
	bootstrapSecret, err := GetBootstrapSecret(kubeClient)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", string(bootstrapSecret.Data["token-id"]), string(bootstrapSecret.Data["token-secret"])), nil
}

func GetBootstrapSecretFromSA(
	kubeClient kubernetes.Interface) (*corev1.Secret, error) {
	sa, err := kubeClient.CoreV1().
		ServiceAccounts(config.OpenClusterManagementNamespace).
		Get(context.TODO(), config.BootstrapSAName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var secret *corev1.Secret
	prefix := config.BootstrapSAName
	if len(prefix) > 63 {
		prefix = prefix[:37]
	}
	for _, objectRef := range sa.Secrets {
		if objectRef.Namespace != "" && objectRef.Namespace != config.OpenClusterManagementNamespace {
			continue
		}
		if strings.HasPrefix(objectRef.Name, prefix) {
			secret, err = kubeClient.CoreV1().
				Secrets(config.OpenClusterManagementNamespace).
				Get(context.TODO(), objectRef.Name, metav1.GetOptions{})
			if err != nil {
				continue
			}
			if secret.Type == corev1.SecretTypeServiceAccountToken {
				break
			}
		}
	}
	if secret == nil {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    corev1.GroupName,
			Resource: "secrets"},
			fmt.Sprintf("secret with prefix %s and type %s not found in service account %s/%s",
				prefix,
				corev1.SecretTypeServiceAccountToken,
				config.OpenClusterManagementNamespace,
				config.BootstrapSAName))
	}
	return secret, nil
}

//GetBootstrapSecretFromSA retrieves the service-account token secret
func GetBootstrapTokenFromSA(
	kubeClient kubernetes.Interface) (string, error) {
	secret, err := GetBootstrapSecretFromSA(kubeClient)
	if err != nil {
		return "", err
	}
	return string(secret.Data["token"]), nil
}

//IsClusterManagerInstalled checks if the hub is already initialized.
//It checks if the crd is already present to find out that the hub is already initialized.
func IsClusterManagerInstalled(apiExtensionsClient apiextensionsclient.Interface) (bool, error) {
	_, err := apiExtensionsClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "clustermanagers.operator.open-cluster-management.io", metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
	}
	return false, err
}

// IsKlusterlets checks if the Managed cluster is already initialized.
//It checks if the crd is already present to find out that the managed cluster is already initialized.
func IsKlusterletsInstalled(apiExtensionsClient apiextensionsclient.Interface) (bool, error) {
	_, err := apiExtensionsClient.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(context.TODO(), "klusterlets.operator.open-cluster-management.io", v1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
	}
	return false, err
}

// WatchUntil starts a watch stream and holds until the condition is satisfied.
func WatchUntil(
	watchFunc func() (watch.Interface, error),
	assertEvent func(event watch.Event) bool) error {
	w, err := watchFunc()
	if err != nil {
		return err
	}
	defer w.Stop()
	for {
		event, ok := <-w.ResultChan()
		if !ok { //The channel is closed by Kubernetes, thus, user should check the pod status manually
			return fmt.Errorf("unexpected watch event received")
		}

		if assertEvent(event) {
			break
		}
	}
	return nil
}
