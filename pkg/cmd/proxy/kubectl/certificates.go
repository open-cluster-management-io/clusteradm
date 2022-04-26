// Copyright Contributors to the Open Cluster Management project
package kubectl

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
	inClusterSecretServer  = "proxy-server"
	inClusterSecretClient  = "proxy-client"
)

type proxyCertificates struct {
	ca         []byte
	serverCert []byte
	serverKey  []byte
	clientCert []byte
	clientKey  []byte
}

func getProxyCertificates(hubRestConfig *rest.Config, proxyConfig *proxyv1alpha1.ManagedProxyConfiguration) (*proxyCertificates, error) {
	nativeClient, err := kubernetes.NewForConfig(hubRestConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building cilent")
	}

	pc := &proxyCertificates{}

	// ca
	caSecret, err := nativeClient.CoreV1().Secrets(proxyConfig.Spec.ProxyServer.Namespace).
		Get(context.TODO(), inClusterSecretProxyCA, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting CA secret")
	}
	pc.ca = caSecret.Data["ca.crt"]

	// server
	serverCertSecret, err := nativeClient.CoreV1().Secrets(proxyConfig.Spec.ProxyServer.Namespace).
		Get(context.TODO(), inClusterSecretServer, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting cert & key secret")
	}
	pc.serverCert = serverCertSecret.Data["tls.crt"]
	pc.serverKey = serverCertSecret.Data["tls.key"]

	// client
	certSecret, err := nativeClient.CoreV1().Secrets(proxyConfig.Spec.ProxyServer.Namespace).
		Get(context.TODO(), inClusterSecretClient, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting cert & key secret")
	}
	pc.clientCert = certSecret.Data["tls.crt"]
	pc.clientKey = certSecret.Data["tls.key"]

	return pc, nil
}

func buildTLSConfig(caData, certData, keyData []byte, serverName string, protos []string) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caData)
	cert, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}

	tlsCfg := &tls.Config{
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ServerName:   serverName,
	}
	if len(protos) > 0 {
		tlsCfg.NextProtos = protos
	}
	return tlsCfg, nil
}
