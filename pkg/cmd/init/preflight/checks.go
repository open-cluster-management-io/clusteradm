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
	Config clientcmd.ClientConfig
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
	config, err := c.Config.RawConfig()
	if err != nil {
		return nil, []error{err}
	}
	cluster, err := loadCurrentCluster(config)
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
	Config       clientcmd.ClientConfig
	Client       kubernetes.Interface
}

func (c ClusterInfoCheck) Check() (warnings []string, errorList []error) {
	cm, err := c.Client.CoreV1().ConfigMaps(c.Namespace).Get(context.Background(), c.ResourceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resourceNotFound := errors.New("no ConfigMap named cluster-info in the kube-public namespace, clusteradm will creates it")
			config, err := c.Config.RawConfig()
			if err != nil {
				return nil, []error{err}
			}
			cluster, err := loadCurrentCluster(config)
			if err != nil {
				return nil, []error{err}
			}
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
func loadCurrentCluster(currentConfig clientcmdapi.Config) (*api.Cluster, error) {
	context := currentConfig.CurrentContext
	currentCtx, exists := currentConfig.Contexts[context]
	if !exists {
		return nil, errors.Errorf("failed to find the given Current Context in Contexts of the kubeconfig")
	}
	currentCluster, exists := currentConfig.Clusters[currentCtx.Cluster]
	if !exists {
		return nil, errors.Errorf("failed to find the given CurrentContext Cluster in Clusters of the kubeconfig")
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
