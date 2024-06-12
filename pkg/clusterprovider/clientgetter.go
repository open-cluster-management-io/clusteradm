// Copyright Contributors to the Open Cluster Management project
package clusterprovider

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type CachedClientGetter struct {
	config clientcmd.ClientConfig
}

func NewCachedClientGetter(data []byte) (*CachedClientGetter, error) {
	config, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return nil, err
	}
	return &CachedClientGetter{
		config: config,
	}, nil
}

func (c CachedClientGetter) ToRESTConfig() (*rest.Config, error) {
	config, err := c.config.ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (c CachedClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(config)
	return memory.NewMemCacheClient(discoveryClient), nil
}

// ToRESTMapper returns a restmapper
func (c CachedClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}

// ToRawKubeConfigLoader return kubeconfig loader as-is
func (c CachedClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return c.config
}
