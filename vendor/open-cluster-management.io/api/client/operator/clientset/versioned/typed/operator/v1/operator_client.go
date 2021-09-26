// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	rest "k8s.io/client-go/rest"
	"open-cluster-management.io/api/client/operator/clientset/versioned/scheme"
	v1 "open-cluster-management.io/api/operator/v1"
)

type OperatorV1Interface interface {
	RESTClient() rest.Interface
	ClusterManagersGetter
	KlusterletsGetter
}

// OperatorV1Client is used to interact with features provided by the operator.open-cluster-management.io group.
type OperatorV1Client struct {
	restClient rest.Interface
}

func (c *OperatorV1Client) ClusterManagers() ClusterManagerInterface {
	return newClusterManagers(c)
}

func (c *OperatorV1Client) Klusterlets() KlusterletInterface {
	return newKlusterlets(c)
}

// NewForConfig creates a new OperatorV1Client for the given config.
func NewForConfig(c *rest.Config) (*OperatorV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &OperatorV1Client{client}, nil
}

// NewForConfigOrDie creates a new OperatorV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *OperatorV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new OperatorV1Client for the given RESTClient.
func New(c rest.Interface) *OperatorV1Client {
	return &OperatorV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *OperatorV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
