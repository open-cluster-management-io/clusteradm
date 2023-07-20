// Copyright Contributors to the Open Cluster Management project
package preflight

import (
	"context"
	"net"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var BootstrapConfigMap = "cluster-info"

type SingletonControlplaneCheck struct {
	ControlplaneName string
}

func (c SingletonControlplaneCheck) Check() (warnings []string, errorList []error) {
	re := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	matched := re.MatchString(c.ControlplaneName)
	if !matched {
		return nil, []error{errors.New("validate ControlplaneName failed: should match `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`")}
	}
	return nil, nil
}

func (c SingletonControlplaneCheck) Name() string {
	return "SingletonControlplane check"
}

type HubApiServerCheck struct {
	ClusterCtx string // current-context in kubeconfig
	ConfigPath string // kubeconfig file path
}

func checkServer(server string) (warnings []string, errorList []error) {
	u, err := url.Parse(server)
	if err != nil {
		return nil, []error{err}
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		missingPortInAddressErr := net.AddrError{Err: "missing port in address", Addr: u.Host}
		if err.Error() == missingPortInAddressErr.Error() {
			host = u.Host
		} else {
			return nil, []error{err}
		}
	}
	if net.ParseIP(host) == nil {
		return []string{"Hub Api Server is a domain name, maybe you should set HostAlias in klusterlet"}, nil
	}
	return nil, nil
}

func (c HubApiServerCheck) Check() (warnings []string, errorList []error) {
	cluster, err := loadCurrentCluster(c.ClusterCtx, c.ConfigPath)
	if err != nil {
		return nil, []error{err}
	}
	return checkServer(cluster.Server)
}

func (c HubApiServerCheck) Name() string {
	return "HubApiServer check"
}

// ClusterInfoCheck checks whether the target kubernetes resource exist in the cluster.
type ClusterInfoCheck struct {
	Namespace    string
	ResourceName string
	ClusterCtx   string // current-context in kubeconfig
	ConfigPath   string // kubeconfig file path
	Client       kubernetes.Interface
}

func (c ClusterInfoCheck) Check() (warnings []string, errorList []error) {
	cm, err := c.Client.CoreV1().ConfigMaps(c.Namespace).Get(context.Background(), c.ResourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resourceNotFound := errors.New("no ConfigMap named cluster-info in the kube-public namespace, clusteradm will creates it")
			cluster, err := loadCurrentCluster(c.ClusterCtx, c.ConfigPath)
			if err != nil {
				return []string{resourceNotFound.Error()}, []error{err}
			}
			if err := createClusterInfo(c.Client, cluster); err != nil {
				return []string{resourceNotFound.Error()}, []error{err}
			}
			return []string{resourceNotFound.Error()}, nil
		}
		return nil, []error{err}
	}
	if len(cm.Data["kubeconfig"]) == 0 {
		return nil, []error{errors.New("empty kubeconfig data in cluster-info")}
	}
	return nil, nil
}

func (c ClusterInfoCheck) Name() string {
	return "cluster-info check"
}

// loadCurrentCluster will load kubeconfig from file and return the current cluster.
// The default file path is ~/.kube/config.
func loadCurrentCluster(context string, kubeConfigFilePath string) (*api.Cluster, error) {
	var (
		currentConfig *clientcmdapi.Config
		err           error
	)

	if len(kubeConfigFilePath) == 0 {
		currentConfig, err = clientcmd.NewDefaultClientConfigLoadingRules().Load()
		if err != nil {
			return nil, err
		}
	} else {
		currentConfig, err = clientcmd.LoadFromFile(kubeConfigFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load kubeconfig file %s that already exists on disk", kubeConfigFilePath)
		}
	}
	// load kubeconfig from file
	// get the hub cluster context
	if len(context) == 0 {
		// use the current context from the kubeconfig
		context = currentConfig.CurrentContext
	}
	currentCtx, exists := currentConfig.Contexts[context]
	if !exists {
		return nil, errors.Errorf("failed to find the given Current Context in Contexts of the kubeconfig file %s", kubeConfigFilePath)
	}
	currentCluster, exists := currentConfig.Clusters[currentCtx.Cluster]
	if !exists {
		return nil, errors.Errorf("failed to find the given CurrentContext Cluster in Clusters of the kubeconfig file %s", kubeConfigFilePath)
	}
	return currentCluster, nil
}

// createClusterInfo will create a ConfigMap named cluster-info in the kube-public namespace.
func createClusterInfo(client kubernetes.Interface, cluster *clientcmdapi.Cluster) error {
	kubeconfig := &clientcmdapi.Config{Clusters: map[string]*clientcmdapi.Cluster{"": cluster}}
	if err := clientcmdapi.FlattenConfig(kubeconfig); err != nil {
		return err
	}
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return err
	}
	clusterInfo := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      BootstrapConfigMap,
			Namespace: metav1.NamespacePublic,
		},
		Immutable: BoolPointer(true),
		Data: map[string]string{
			"kubeconfig": string(kubeconfigBytes),
		},
	}
	return CreateOrUpdateConfigMap(client, clusterInfo)
}
