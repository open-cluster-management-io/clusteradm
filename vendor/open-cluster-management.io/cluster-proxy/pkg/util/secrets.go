package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	proxyv1alpha1 "open-cluster-management.io/cluster-proxy/pkg/apis/proxy/v1alpha1"
)

const (
	inClusterSecretProxyCA = "proxy-server-ca"
	inClusterSecretClient  = "proxy-client"
)

func GetKonnectivityTLSConfig(restConfig *rest.Config, proxyConfig *proxyv1alpha1.ManagedProxyConfiguration) (*tls.Config, error) {
	// building tls config from secret data
	nativeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building cilent")
	}
	caSecret, err := nativeClient.CoreV1().Secrets(proxyConfig.Spec.ProxyServer.Namespace).
		Get(context.TODO(), inClusterSecretProxyCA, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting CA secret")
	}
	caData := caSecret.Data["ca.crt"]
	certSecret, err := nativeClient.CoreV1().Secrets(proxyConfig.Spec.ProxyServer.Namespace).
		Get(context.TODO(), inClusterSecretClient, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting cert & key secret")
	}
	certData := certSecret.Data["tls.crt"]
	keyData := certSecret.Data["tls.key"]

	// building tls config from pem-encoded data
	tlsCfg, err := buildTLSConfig(
		caData,
		certData,
		keyData,
		proxyConfig.Spec.ProxyServer.InClusterServiceName+"."+proxyConfig.Spec.ProxyServer.Namespace,
		nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building TLS config from secret")
	}
	return tlsCfg, nil

}

func buildTLSConfig(caData, certData, keyData []byte, serverName string, protos []string) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	clientCert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{clientCert},
		ServerName:   serverName,
	}
	if len(protos) > 0 {
		tlsCfg.NextProtos = protos
	}
	return tlsCfg, nil
}
