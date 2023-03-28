// Copyright Contributors to the Open Cluster Management project
package kubectl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1"
	proxyv1alpha1 "open-cluster-management.io/cluster-proxy/pkg/apis/proxy/v1alpha1"
	"open-cluster-management.io/cluster-proxy/pkg/common"
	clusterproxyclient "open-cluster-management.io/cluster-proxy/pkg/generated/clientset/versioned"
	"open-cluster-management.io/cluster-proxy/pkg/util"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	msaClientv1alpha1 "open-cluster-management.io/managed-serviceaccount/pkg/generated/clientset/versioned"
)

func NewCmd(clusteradmFlags *genericclioptionsclusteradm.ClusteradmFlags, streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(clusteradmFlags)

	var hubRestConfig *rest.Config
	var proxyConfig *proxyv1alpha1.ManagedProxyConfiguration

	cmd := &cobra.Command{
		Use:   "kubectl",
		Short: "Use kubectl through cluster-proxy addon.",
		Long:  "Use kubectl through cluster-proxy addon. (Only supports managed service account token as certificate.)",
		Example: `If you want to get nodes on managed cluster named "cluster1", you can use the following command:
		clusteradm proxy kubectl --cluster=cluster1 --sa=test --args="get nodes"`,
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if err = o.validate(); err != nil {
				return err
			}

			// get hubRestConfig
			hubRestConfig, err = o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
			if err != nil {
				return errors.Wrapf(err, "failed loading hub cluster's client config")
			}

			// get proxyConfig
			proxyConfig, err = getProxyConfig(hubRestConfig, streams)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get managedcluster
			clusterClient, err := clusterv1.NewForConfig(hubRestConfig)
			if err != nil {
				return err
			}
			_, err = clusterClient.ManagedClusters().Get(context.TODO(), o.ClusterOption.Cluster, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// Get managedServiceAccount
			managedServiceAccountToken, err := getManagedServiceAccountToken(hubRestConfig, o.managedServiceAccount, o.ClusterOption.Cluster)
			if err != nil {
				return err
			}

			// Get Proxy Certificates
			proxyCertificates, err := getProxyCertificates(hubRestConfig, proxyConfig)
			if err != nil {
				return err
			}

			// Run port-forward in goroutine
			localProxy := util.NewRoundRobinLocalProxy(
				hubRestConfig,
				proxyConfig.Spec.ProxyServer.Namespace,
				common.LabelKeyComponentName+"="+common.ComponentNameProxyServer,
				int32(8090), // TODO make it configurable or random later
			)
			portForwardClose, err := localProxy.Listen()
			if err != nil {
				return errors.Wrapf(err, "failed listening local proxy")
			}
			defer portForwardClose()

			// Run a http-proxy-server in goroutine
			hps, err := newHttpProxyServer(
				cmd.Context(),
				o.ClusterOption.Cluster,
				int32(8090), // TODO make it configurable or random later
				proxyCertificates,
			)
			if err != nil {
				return err
			}
			err = hps.Listen(cmd.Context(), int32(9090)) // TODO make it configurable or random later
			if err != nil {
				return errors.Wrapf(err, "failed listening http proxy server")
			}

			// Configure a customized kubeconfig amd write into /tmp dir with a random name
			tmpKubeconfigFilePath, err := genTmpKubeconfig(o.ClusterOption.Cluster, managedServiceAccountToken)
			if err != nil {
				return err
			}
			defer os.Remove(tmpKubeconfigFilePath)
			klog.V(4).Infof("kubeconfig file path is %s", tmpKubeconfigFilePath)

			// Using kubectl to access the managed cluster using the above customized kubeconfig
			// We are using combinedoutput, so err msg should include in result, no need to handle err
			result, _ := runKubectlCommand(tmpKubeconfigFilePath, o.kubectlArgs)
			if _, err = streams.Out.Write(result); err != nil {
				return errors.Wrap(err, "streams out write failed")
			}

			return nil
		},
	}

	o.ClusterOption.AddFlags(cmd.Flags())
	cmd.Flags().StringVar(&o.managedServiceAccount, "sa", "", "The name of the managedServiceAccount")
	cmd.Flags().StringVar(&o.kubectlArgs, "args", "", "The arguments to pass to kubectl")

	return cmd
}

func getProxyConfig(hubRestConfig *rest.Config, streams genericclioptions.IOStreams) (*proxyv1alpha1.ManagedProxyConfiguration, error) {
	addonClient, err := addonv1alpha1client.NewForConfig(hubRestConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed initializing addon api client")
	}

	clusterAddon, err := addonClient.AddonV1alpha1().ClusterManagementAddOns().Get(
		context.TODO(),
		"cluster-proxy",
		metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := fmt.Fprintf(
				streams.Out,
				"Cluster-Proxy addon is not installed.\n"); err != nil {
				return nil, err
			}
			if _, err := fmt.Fprintf(
				streams.Out,
				"Consider following the guide: https://open-cluster-management.io/getting-started/integration/cluster-proxy/\n"); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed checking cluster management addon for cluster-proxy")
	}

	proxyClient, err := clusterproxyclient.NewForConfig(hubRestConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed initializing proxy api client")
	}

	// TODO: fix this deprecated field AddOnConfiguration
	// nolint:staticcheck
	proxyConfig, err := proxyClient.ProxyV1alpha1().ManagedProxyConfigurations().
		Get(context.TODO(), clusterAddon.Spec.AddOnConfiguration.CRName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting managedproxyconfiguration for cluster-proxy")
	}

	return proxyConfig, nil
}

func getManagedServiceAccountToken(hubRestConfig *rest.Config, msaName string, namespace string) (string, error) {
	msaClient, err := msaClientv1alpha1.NewForConfig(hubRestConfig)
	if err != nil {
		return "", err
	}

	msa, err := msaClient.Authentication().ManagedServiceAccounts(namespace).Get(context.TODO(), msaName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	kubeClient, err := kubernetes.NewForConfig(hubRestConfig)
	if err != nil {
		return "", err
	}
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), msa.Status.TokenSecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", errors.Errorf("token is not found in secret %s", secret.Name)
	}

	return string(token), nil
}

// Configure a tmp kubeconfig and store it in a tmp file
func genTmpKubeconfig(cluster string, msaToken string) (string, error) {
	c := &clientcmdapi.Cluster{
		Server:                "https://localhost:9090",
		InsecureSkipTLSVerify: true, // Because we are using a local proxy
	}

	kubeconfigContent, err := clientcmd.Write(clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": c,
		},
		Contexts: map[string]*clientcmdapi.Context{
			"context": {
				Cluster:  "cluster",
				AuthInfo: "user",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {
				Token: msaToken,
			},
		},
		CurrentContext: "context",
	})
	if err != nil {
		return "", nil
	}

	tmpFile := fmt.Sprintf("/tmp/%s-%s.kubeconfig", cluster, uuid.New().String())
	if err := os.WriteFile(tmpFile, kubeconfigContent, 0644); err != nil {
		return "", err
	}

	return tmpFile, nil
}

// Use runKubectlCommand to access the managed cluster
func runKubectlCommand(kubeconfigFilePath string, args string) ([]byte, error) {
	cmd := exec.Command("kubectl", strings.Split(args, " ")...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfigFilePath))
	return cmd.CombinedOutput()
}
