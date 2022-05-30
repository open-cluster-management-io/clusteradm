// Copyright Contributors to the Open Cluster Management project
package kubectl

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
	konnectivity "sigs.k8s.io/apiserver-network-proxy/konnectivity-client/pkg/client"
)

type httpProxyServer struct {
	getTunnel       func() (konnectivity.Tunnel, error)
	serverTLSConfig *tls.Config
	cluster         string
}

func newHttpProxyServer(
	ctx context.Context,
	cluster string,
	proxyServerPort int32,
	pc *proxyCertificates,
) (*httpProxyServer, error) {
	// build client tls config, using to access proxy-server
	proxyClientTLSCfg, err := buildTLSConfig(pc.ca, pc.clientCert, pc.clientKey, "localhost", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building TLS config from secret")
	}

	// build server tls config and use proxyServer's tls to start this http-proxyserver as well
	proxyServerTLSCfg, err := buildTLSConfig(pc.ca, pc.serverCert, pc.serverKey, "localhost", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed building TLS config from secret")
	}

	return &httpProxyServer{
		getTunnel: func() (konnectivity.Tunnel, error) {
			// instantiate a gprc proxy dialer
			tunnel, err := konnectivity.CreateSingleUseGrpcTunnel(
				ctx,
				net.JoinHostPort("localhost", strconv.Itoa(int(proxyServerPort))),
				grpc.WithTransportCredentials(grpccredentials.NewTLS(proxyClientTLSCfg)),
				grpc.WithKeepaliveParams(keepalive.ClientParameters{
					Time: time.Second * 5,
				}),
			)
			if err != nil {
				return nil, err
			}
			return tunnel, nil
		},
		serverTLSConfig: proxyServerTLSCfg,
		cluster:         cluster,
	}, nil
}

func (s *httpProxyServer) Listen(ctx context.Context, port int32) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)

	srv := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		Handler:   mux,
		TLSConfig: s.serverTLSConfig,
	}
	go func() {
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			runtime.HandleError(errors.Wrapf(err, "failed to listen http proxy server"))
		}
	}()
	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(ctx); err != nil {
			runtime.HandleError(errors.Wrapf(err, "failed to shutdown http proxy server"))
		}
	}()
	return nil
}

func (s *httpProxyServer) handle(wr http.ResponseWriter, req *http.Request) {
	if klog.V(4).Enabled() {
		dump, err := httputil.DumpRequest(req, true)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadRequest)
			return
		}
		klog.V(4).Infof("request:\n%s", string(dump))
	}

	target := fmt.Sprintf("https://%s", s.cluster)
	apiserverURL, err := url.Parse(target)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	tunnel, err := s.getTunnel()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	var proxyConn net.Conn
	defer func() {
		if proxyConn != nil {
			err = proxyConn.Close()
			if err != nil {
				klog.V(4).ErrorS(err, "connection closed")
			}
		}
	}()

	proxy := httputil.NewSingleHostReverseProxy(apiserverURL)
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip server-auth for kube-apiserver
		},
		// golang http pkg automatically upgrade http connection to http2 connection, but http2 can not upgrade to SPDY which used in "kubectl exec".
		// set ForceAttemptHTTP2 = false to prevent auto http2 upgration
		ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// TODO: may find a way to cache the proxyConn.
			proxyConn, err = tunnel.DialContext(ctx, network, addr)
			return proxyConn, err
		},
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, e error) {
		_, err = rw.Write([]byte(fmt.Sprintf("proxy to anp-proxy-server failed because %v", e)))
		if err != nil {
			klog.Errorf("response write fail %v", e)
			return
		}
		klog.Errorf("proxy to anp-proxy-server failed because %v", e)
	}

	klog.V(4).Infof("request scheme:%s; rawQuery:%s; path:%s", req.URL.Scheme, req.URL.RawQuery, req.URL.Path)

	proxy.ServeHTTP(wr, req)
}
