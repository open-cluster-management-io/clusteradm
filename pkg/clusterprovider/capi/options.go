// Copyright Contributors to the Open Cluster Management project
package capi

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/clusterprovider"
)

var capiGVR = schema.GroupVersionResource{
	Group:    "cluster.x-k8s.io",
	Version:  "v1beta1",
	Resource: "clusters",
}

type CAPIOptions struct {
	// KubeConfigFile is the kubeconfig to connect to capi management cluster
	KubeConfigFile string
	// name of the cluster
	ClusterName      string
	ClusterNamespace string

	Enable bool

	f cmdutil.Factory
}

func NewCAPIOption(factory cmdutil.Factory) *CAPIOptions {
	return &CAPIOptions{
		f: factory,
	}
}

func (o *CAPIOptions) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.KubeConfigFile, "capi-kubeconfig", "", "kubeconfig to connect to capi management cluster.")
	flags.StringVar(&o.ClusterName, "capi-cluster-name", "", "cluster name of capi to join, in the format of namespace/name")
	flags.BoolVar(&o.Enable, "capi-import", false, "impor from capi, capi-cluster-name must be set when this is set to true")
}

func (o *CAPIOptions) Complete(cmd *cobra.Command, args []string) (err error) {
	if o.Enable {
		capiNamespace, capiName, err := cache.SplitMetaNamespaceKey(o.ClusterName)
		if err != nil {
			return err
		}

		if len(capiNamespace) == 0 {
			capiNamespace = metav1.NamespaceDefault
		}
		o.ClusterNamespace = capiNamespace
		o.ClusterName = capiName
	}
	return nil
}

func (o *CAPIOptions) Validate() error {
	if o.Enable && len(o.ClusterName) == 0 {
		return fmt.Errorf("capi-cluster-name must be set")
	}
	return nil
}

func (o *CAPIOptions) ToClientGetter() (genericclioptions.RESTClientGetter, error) {
	var config *rest.Config
	var err error
	if len(o.KubeConfigFile) == 0 {
		config, err = o.f.ToRESTConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", o.KubeConfigFile)
		if err != nil {
			return nil, err
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	_, err = dynamicClient.Resource(capiGVR).Namespace(o.ClusterNamespace).Get(context.TODO(), o.ClusterName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s/%s from capi, got error %v", o.ClusterNamespace, o.ClusterName, err)
	}

	secret, err := kubeClient.CoreV1().Secrets(o.ClusterNamespace).Get(context.TODO(), o.ClusterName+"-kubeconfig", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, fmt.Errorf("kubeconfig sercret for cluster %s/%s is not found, try again later", o.ClusterNamespace, o.ClusterName)
	}
	if err != nil {
		return nil, err
	}

	data, ok := secret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("missing key %q in secret data", o.ClusterName)
	}
	return clusterprovider.NewCachedClientGetter(data)
}
