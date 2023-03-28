// Copyright Contributors to the Open Cluster Management project
package health

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8snet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1"
	proxyv1alpha1 "open-cluster-management.io/cluster-proxy/pkg/apis/proxy/v1alpha1"
	"open-cluster-management.io/cluster-proxy/pkg/common"
	"open-cluster-management.io/cluster-proxy/pkg/generated/clientset/versioned"
	"open-cluster-management.io/cluster-proxy/pkg/util"
	konnectivity "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/client"
	proxyutil "sigs.k8s.io/apiserver-network-proxy/pkg/util"
)

func (o *Options) complete(cmd *cobra.Command, args []string) error {
	if len(o.proxyClientCACertPath) > 0 && len(o.proxyClientCertPath) > 0 && len(o.proxyClientKeyPath) > 0 {
		o.isProxyServerAddressProvided = true
	}
	if o.proxyServerHost != addrLocalhost {
		o.isProxyServerAddressProvided = true
	}
	return nil
}

func (o *Options) validate() error {
	if !o.inClusterProxyCertLookup {
		if len(o.proxyClientCACertPath) == 0 {
			return errors.New("--proxy-ca-cert must be set when in-cluster lookup is disabled")
		}
		if len(o.proxyClientCertPath) == 0 {
			return errors.New("--proxy-cert must be set when in-cluster lookup is disabled")
		}
		if len(o.proxyClientKeyPath) == 0 {
			return errors.New("--proxy-key must be set when in-cluster lookup is disabled")
		}
	}
	if err := o.ClusterOption.Validate(); err != nil {
		return err
	}
	return nil
}

func (o *Options) run(streams genericclioptions.IOStreams) error {

	hubRestConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return errors.Wrapf(err, "failed loading hub cluster's client config")
	}
	addonClient, err := addonv1alpha1client.NewForConfig(hubRestConfig)
	if err != nil {
		return errors.Wrapf(err, "failed initializing addon api client")
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
				return err
			}
			if _, err := fmt.Fprintf(
				streams.Out,
				"Consider following the guide: https://open-cluster-management.io/getting-started/integration/cluster-proxy/\n"); err != nil {
				return err
			}
			return nil
		}
		return errors.Wrapf(err, "failed checking cluster management addon for cluster-proxy")
	}

	proxyClient, err := versioned.NewForConfig(hubRestConfig)
	if err != nil {
		return errors.Wrapf(err, "failed initializing proxy api client")
	}

	// TODO: fix this deprecated field AddOnConfiguration
	// nolint:staticcheck
	proxyConfig, err := proxyClient.ProxyV1alpha1().ManagedProxyConfigurations().
		Get(context.TODO(), clusterAddon.Spec.AddOnConfiguration.CRName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed getting managedproxyconfiguration for cluster-proxy")
	}

	clusterClient, err := clusterv1.NewForConfig(hubRestConfig)
	if err != nil {
		return errors.Wrapf(err, "failed initializing cluster client")
	}
	managedClusterList, err := clusterClient.ManagedClusters().List(
		context.TODO(),
		metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed listing managed clusters")
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	if !o.isProxyServerAddressProvided {
		// run a local port-forward server if no proxy server specified
		localProxy := util.NewRoundRobinLocalProxy(
			hubRestConfig,
			proxyConfig.Spec.ProxyServer.Namespace,
			common.LabelKeyComponentName+"="+common.ComponentNameProxyServer, // TODO: configurable label selector?
			int32(o.proxyServerPort),
		)
		closeFn, err := localProxy.Listen()
		if err != nil {
			return errors.Wrapf(err, "failed listening local proxy")
		}
		defer closeFn()
	}

	tlsCfg, err := o.getKonnectivityTLSConfig(proxyConfig)
	if err != nil {
		return errors.Wrapf(err, "failed building tls config")
	}

	tunnel, err := konnectivity.CreateSingleUseGrpcTunnel(
		ctx,
		net.JoinHostPort(o.proxyServerHost, strconv.Itoa(o.proxyServerPort)),
		grpc.WithTransportCredentials(grpccredentials.NewTLS(tlsCfg)),
	)
	if err != nil {
		return errors.Wrapf(err, "failed starting konnectivity proxy")
	}

	probingClusters := o.ClusterOption.AllClusters()
	w := newWriter(streams)
	for _, cluster := range managedClusterList.Items {
		if probingClusters.Len() == 0 || probingClusters.Has(cluster.Name) {
			if err := o.visit(&w, hubRestConfig, addonClient, tunnel.DialContext, cluster.Name); err != nil {
				klog.Errorf("An error occurred when requesting: %v", err)
			}
		}
	}

	w.flush()
	return nil
}

const (
	inClusterSecretProxyCA = "proxy-server-ca"
	inClusterSecretClient  = "proxy-client"
)

func (o *Options) getKonnectivityTLSConfig(proxyConfig *proxyv1alpha1.ManagedProxyConfiguration) (*tls.Config, error) {
	if o.isProxyClientCertProvided {
		// building tls config from local paths
		tlsCfg, err := proxyutil.GetClientTLSConfig(
			o.proxyClientCACertPath,
			o.proxyClientCertPath,
			o.proxyClientKeyPath,
			o.proxyServerHost, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "failed building TLS config for konnectivity client")
		}
		return tlsCfg, nil
	}
	// building tls config from secret data
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed building cilent config")
	}
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
	tlsCfg, err := buildTLSConfig(caData, certData, keyData, o.proxyServerHost, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building TLS config from secret")
	}
	return tlsCfg, nil

}

func (o *Options) visit(
	w *writer,
	hubRestConfig *rest.Config,
	addonClient addonv1alpha1client.Interface,
	dialFunc k8snet.DialFunc,
	clusterName string) error {

	addon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).
		Get(context.TODO(), common.AddonName, metav1.GetOptions{})
	installed := "True"
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed getting managed cluster addon for cluster %v", clusterName)
		}
		installed = "False"
	}
	addonAvailable := "False"
	if meta.IsStatusConditionTrue(addon.Status.Conditions, addonv1alpha1.ManagedClusterAddOnConditionAvailable) {
		addonAvailable = "True"
	}

	copiedCfg := rest.CopyConfig(hubRestConfig)
	copiedCfg.Dial = dialFunc
	copiedCfg.TLSClientConfig = rest.TLSClientConfig{}
	// For now we have to use an insecure tls config here because the konnectivity
	// tunnel is routing based on the name of the managed cluster, and the proxied
	// requests reaching the target managed cluster will be using the managed
	// cluster's name as the hostname. If the target cluster's kube-apiserver
	// doesn't sign the managed cluster name into its server certificate's SAN, the
	// client will be failing due to server hostname validation.
	// TODO: Securing the proxied requests.
	copiedCfg.TLSClientConfig.Insecure = true

	rt, err := rest.TransportFor(copiedCfg)
	if err != nil {
		return errors.Wrapf(err, "failed creating roundtripper for cluster %v", clusterName)
	}
	req := &http.Request{
		Method: "GET",
		Host:   clusterName,
		URL: &url.URL{
			Scheme: "https",
			Host:   clusterName,
			Path:   "/healthz",
		},
	}
	health := "False"
	latency := "<none>"
	start := time.Now()
	resp, err := rt.RoundTrip(req)
	if err != nil {
		health = "Unknown"
		klog.Errorf("Failed requesting /healthz endpoint for cluster %v: %v", clusterName, err)
		if strings.Contains(err.Error(), "dial timeout") {
			latency = "<timeout>"
		}
		w.print(clusterName, installed, addonAvailable, health, latency)
		return nil
	}

	end := time.Now()
	data, _ := io.ReadAll(resp.Body)
	if "ok" == string(data) {
		health = "True"
		latency = end.Sub(start).String()
	}

	// TODO: use a common table convertor in the future.
	w.print(clusterName, installed, addonAvailable, health, latency)
	return nil
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

type writer struct {
	w *tabwriter.Writer
}

func newWriter(streams genericclioptions.IOStreams) writer {
	w := tabwriter.NewWriter(streams.Out, 4, 8, 4, ' ', 0)
	// header
	_, _ = fmt.Fprintf(w,
		"%s\t%s\t%s\t%s\t%s\n",
		"CLUSTER NAME",
		"INSTALLED",
		"AVAILABLE",
		"PROBED HEALTH",
		"LATENCY",
	)
	return writer{w}
}

func (w *writer) print(clusterName, installed, available, health, latency string) {
	_, _ = fmt.Fprintf(w.w,
		"%s\t%s\t%s\t%s\t%s\n",
		clusterName, installed, available, health, latency,
	)
}

func (w *writer) flush() {
	_ = w.w.Flush()
}
