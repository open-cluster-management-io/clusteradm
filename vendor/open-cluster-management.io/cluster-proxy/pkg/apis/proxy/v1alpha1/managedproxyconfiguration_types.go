/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ManagedProxyConfiguration{}, &ManagedProxyConfigurationList{})
}

// ManagedProxyConfigurationSpec is the prescription of ManagedProxyConfiguration
type ManagedProxyConfigurationSpec struct {
	// `authentication` defines how the credentials for the authentication
	// between proxy servers and proxy agents are signed and mounted.
	// +required
	Authentication ManagedProxyConfigurationAuthentication `json:"authentication"`
	// `proxyServer` structurelized the arguments for running proxy servers.
	// +required
	ProxyServer ManagedProxyConfigurationProxyServer `json:"proxyServer"`
	// `proxyServer` structurelized the arguments for running proxy agents.
	// +required
	ProxyAgent ManagedProxyConfigurationProxyAgent `json:"proxyAgent"`
	// +optional
	// `deploy` is where we override miscellaneous details for deploying either
	// proxy servers or agents.
	Deploy *ManagedProxyConfigurationDeploy `json:"deploy,omitempty"`
}

// ManagedProxyConfigurationStatus defines the observed state of ManagedProxyConfiguration
type ManagedProxyConfigurationStatus struct {
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// +genclient
// +genclient:nonNamespaced
// ManagedProxyConfiguration is the Schema for the managedproxyconfigurations API
type ManagedProxyConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedProxyConfigurationSpec   `json:"spec,omitempty"`
	Status ManagedProxyConfigurationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedProxyConfigurationList contains a list of ManagedProxyConfiguration
type ManagedProxyConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedProxyConfiguration `json:"items"`
}

// AuthenticationType defines the source of CA certificates, i.e. the signer of both
// proxy servers and agents.
// +kubebuilder:validation:Enum=SelfSigned;Provided;CertManager
type AuthenticationSignerType string

var (
	// `SelfSigned` prescribes the CA certificate and key should be automatically
	// generated if not found in the hub cluster.
	// The self-signed ca key pair will be stored as a secret.
	//
	// Note that the namespace and name can be configured via "--signer-secret-name"
	// and "signer-secret-namespace" at the addon-manager.
	SelfSigned AuthenticationSignerType = "SelfSigned"
	//Provided    AuthenticationType = "Provided"
	//CertManager AuthenticationType = "CertManager"
)

// ManagedProxyConfigurationAuthentication prescribes how we manage the authentication
// between the proxy servers and agents. Overall the authentication are working via
// mTLS certificates so this struct is actually prescribing the signing and storing of
// the managed certificates.
type ManagedProxyConfigurationAuthentication struct {
	// +optional
	// `signer` defines how we sign server and client certificates for the proxy servers
	// and agents.
	Signer ManagedProxyConfigurationCertificateSigner `json:"signer"`
	// +optional
	// `dump` is where we store the signed certificates from signers.
	Dump ManagedProxyConfigurationCertificateDump `json:"dump"`
}

// ManagedProxyConfigurationCertificateSigner prescribes how to sign certificates
// for proxy servers and agents.
type ManagedProxyConfigurationCertificateSigner struct {
	// `type` is the supported type of signer. Currently only "SelfSign" supported.
	// +optional
	// +kubebuilder:default=SelfSigned
	Type AuthenticationSignerType `json:"type"`
	// `selfSigned` prescribes the detail of how we self-sign the certificates.
	// +optional
	SelfSigned *AuthenticationSelfSigned `json:"selfSigned,omitempty"`
}

// ManagedProxyConfigurationCertificateDump prescribes how to dump the signed
// certificates which will be mounted by the instances of proxy servers and agents.
type ManagedProxyConfigurationCertificateDump struct {
	// +optional
	// `secrets` is the names of the secrets for saving the signed certificates.
	Secrets CertificateSigningSecrets `json:"secrets"`
}

// AuthenticationSelfSigned prescribes how to self-sign the certificates.
type AuthenticationSelfSigned struct {
	// +optional
	// `additionalSANs` adds a few custom hostnames or IPs to the signing certificates.
	AdditionalSANs []string `json:"additionalSANs,omitempty"`
}

// CertificateSigningSecrets enumerates the target names of the secrets to be mounted
// onto proxy servers and agents.
type CertificateSigningSecrets struct {
	// `signingProxyServerSecretName` the secret name of the proxy server's listening
	// certificates for serving proxy requests.
	// +kubebuilder:default=proxy-server
	// +optional
	SigningProxyServerSecretName string `json:"signingProxyServerSecretName,omitempty"`
	// `signingProxyClientSecretName` is the secret name for requesting/streaming over
	// the proxy server.
	// +kubebuilder:default=proxy-client
	// +optional
	SigningProxyClientSecretName string `json:"signingProxyClientSecretName,omitempty"`
	// `signingAgentServerSecretName` is the secret name of the proxy servers to receive
	// tunneling handshakes from proxy agents.
	// +kubebuilder:default=agent-server
	// +optional
	SigningAgentServerSecretName string `json:"signingAgentServerSecretName,omitempty"`
}

// ManagedProxyConfigurationDeploy prescribes a few common details for running components.
type ManagedProxyConfigurationDeploy struct {
	// `ports` is the ports for proxying and tunneling.
	Ports ManagedProxyConfigurationDeployPorts `json:"ports"`
}

// NodePlacement describes node scheduling configuration for the pods.
type NodePlacement struct {
	// NodeSelector defines which Nodes the Pods are scheduled on. The default is an empty list.
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations is attached by pods to tolerate any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>.
	// The default is an empty list.
	// +optional
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// ManagedProxyConfigurationDeployPorts is the expected port for wiring up proxy servers
// and agents.
type ManagedProxyConfigurationDeployPorts struct {
	// `proxyServer` is the listening port of proxy server for serving proxy requests.
	// +optional
	// +kubebuilder:default=8090
	ProxyServer int32 `json:"proxyServer"`
	// `agentServer` is the listening port of proxy server for serving tunneling handshakes.
	// +optional
	// +kubebuilder:default=8091
	AgentServer int32 `json:"agentServer"`
	// `healthServer` is for probing the healthiness.
	// +optional
	// +kubebuilder:default=8092
	HealthServer int32 `json:"healthServer"`
	// `adminServer` is the port for debugging and operating.
	// +optional
	// +kubebuilder:default=8095
	AdminServer int32 `json:"adminServer"`
}

// ManagedProxyConfigurationProxyServer prescribes how to deploy proxy servers into
// the hub cluster.
type ManagedProxyConfigurationProxyServer struct {
	// `image` is the container image of the proxy servers.
	// +required
	Image string `json:"image"`
	// `replicas` is the expected replicas of the proxy servers.
	// Note that the replicas will also be reflected in the flag `--server-count`
	// so that agents can discover all the server instances.
	// +kubebuilder:default=3
	// +optional
	Replicas int32 `json:"replicas"`
	// `inClusterServiceName` is the name of the in-cluster service for proxying
	// requests inside the hub cluster to the proxy servers.
	// +optional
	// +kubebuilder:default=proxy-entrypoint
	InClusterServiceName string `json:"inClusterServiceName,omitempty"`
	// `namespace` is the namespace where we will deploy the proxy servers and related
	// resources.
	// +optional
	// +kubebuilder:default=open-cluster-management-cluster-proxy
	Namespace string `json:"namespace,omitempty"`
	// `entrypoint` defines how will the proxy agents connecting the servers.
	// +optional
	Entrypoint *ManagedProxyConfigurationProxyServerEntrypoint `json:"entrypoint"`

	// `additionalArgs` adds arbitrary additional command line args to the proxy-server.
	// +optional
	AdditionalArgs []string `json:"additionalArgs,omitempty"`

	// NodePlacement defines which Nodes the proxy server are scheduled on. The default is an empty list.
	// +optional
	NodePlacement NodePlacement `json:"nodePlacement,omitempty"`
}

// ManagedProxyConfigurationProxyServerEntrypoint prescribes the ingress for serving
// tunneling handshakes from proxy agents.
type ManagedProxyConfigurationProxyServerEntrypoint struct {
	// `type` is the type of the entrypoint of the proxy servers.
	// Currently supports "Hostname", "LoadBalancerService"
	// +required
	Type EntryPointType `json:"type"`
	// `loadBalancerService` points to a load-balancer typed service in the hub cluster.
	// +optional
	LoadBalancerService *EntryPointLoadBalancerService `json:"loadBalancerService,omitempty"`
	// `hostname` points to a fixed hostname for serving agents' handshakes.
	// +optional
	Hostname *EntryPointHostname `json:"hostname,omitempty"`

	// `port` is the target port to access proxy servers
	// +optional
	// +kubebuilder:default=8091
	// +kubebuilder:validation:Minimum=1
	Port int32 `json:"port,omitempty"`
}

// EntryPointType is the type of the entrypoint.
// +kubebuilder:validation:Enum=Hostname;LoadBalancerService;PortForward
type EntryPointType string

var (
	// LoadBalancerService prescribes the proxy agents to establish tunnels via the
	// expose IP from the load-balancer service.
	EntryPointTypeLoadBalancerService EntryPointType = "LoadBalancerService"
	// Hostname prescribes the proxy agents to connect a fixed hostname.
	EntryPointTypeHostname EntryPointType = "Hostname"
	// PortForward prescribes the proxy agent to connect a local proxy served on the
	// addon-agent which proxies tunnel connection to the proxy-servers via pod
	// port-forwarding.
	EntryPointTypePortForward EntryPointType = "PortForward"
)

// EntryPointLoadBalancerService is the reference to a load-balancer service.
type EntryPointLoadBalancerService struct {
	// `name` is the name of the load-balancer service. And the namespace will align
	// to where the proxy-servers are deployed.
	// +optional
	// +kubebuilder:default=proxy-agent-entrypoint
	Name string `json:"name"`

	// Annotations is the annoations of the load-balancer service.
	// This is for allowing customizing service using vendor-specific extended annotations such as:
	// - service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type: "intranet"
	// - service.beta.kubernetes.io/azure-load-balancer-internal: true
	// +optional
	Annotations []AnnotationVar `json:"annotations,omitempty"`
}

// AnnotationVar list of annotation variables to set in the LB Service.
type AnnotationVar struct {
	// Key is the key of annotation
	// +kubebuilder:validation:Required
	// +required
	Key string `json:"key"`

	// Value is the value of annotation
	// +optional
	Value string `json:"value,omitempty"`
}

// EntryPointHostname references a fixed hostname.
type EntryPointHostname struct {
	// +required
	Value string `json:"value"`
}

// ManagedProxyConfigurationProxyAgent prescribes how to deploy agents to the managed
// cluster.
type ManagedProxyConfigurationProxyAgent struct {
	// `image` is the container image of the proxy agent.
	// +required
	Image string `json:"image"`
	// `replicas` is the replicas of the agents.
	// +optional
	// +kubebuilder:default=3
	Replicas int32 `json:"replicas"`
	// `additionalArgs` defines args used in proxy-agent.
	// +optional
	AdditionalArgs []string `json:"additionalArgs,omitempty"`
	// `imagePullSecrets` defines the imagePullSecrets used by proxy-agent
	// +optional
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

const (
	ConditionTypeProxyServerDeployed     = "ProxyServerDeployed"
	ConditionTypeProxyServerSecretSigned = "ProxyServerSecretSigned"
	ConditionTypeAgentServerSecretSigned = "AgentServerSecretSigned"
	ConditionTypeProxyClientSecretSigned = "ProxyClientSecretSigned"
)
