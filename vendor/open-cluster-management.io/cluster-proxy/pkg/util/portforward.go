package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/klog/v2"
)

type LocalProxyServer interface {
	Listen(ctx context.Context) (func(), error)
}

// PortForwardProtocolV1Name is the subprotocol used for port forwarding.
const PortForwardProtocolV1Name = "portforward.k8s.io"

func NewRoundRobinLocalProxy(
	restConfig *rest.Config,
	readiness *atomic.Value,
	podNamespace,
	podSelector string,
	targetPort int32) LocalProxyServer {
	return NewRoundRobinLocalProxyWithReqId(
		restConfig,
		readiness,
		podNamespace,
		podSelector,
		targetPort,
		0,
	)
}

func NewRoundRobinLocalProxyWithReqId(
	restConfig *rest.Config,
	readiness *atomic.Value,
	podNamespace,
	podSelector string,
	targetPort int32,
	reqId int) LocalProxyServer {
	return &roundRobin{
		restConfig:           restConfig,
		proxyServerNamespace: podNamespace,
		podSelector:          podSelector,
		targetPort:           targetPort,
		reqId:                reqId,
		lock:                 &sync.Mutex{},
		firstConnReceived:    readiness,
	}
}

var _ LocalProxyServer = &roundRobin{}

type roundRobin struct {
	proxyServerNamespace string
	podSelector          string
	targetPort           int32

	restConfig        *rest.Config
	lock              *sync.Mutex
	reqId             int
	firstConnReceived *atomic.Value
}

func (r *roundRobin) Listen(ctx context.Context) (func(), error) {
	klog.V(4).Infof("Started local proxy server at port %d", r.targetPort)
	listener, err := net.Listen(
		"tcp",
		net.JoinHostPort("localhost", strconv.Itoa(int(r.targetPort))))
	if err != nil {
		return nil, fmt.Errorf("unable to create listener: Error %s", err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					if !strings.Contains(strings.ToLower(err.Error()), "use of closed network connection") {
						runtime.HandleError(fmt.Errorf("error accepting connection on port %d: %v", r.targetPort, err))
						continue
					}
				}
				if conn != nil {
					go func() {
						if err := r.handle(conn); err != nil {
							runtime.HandleError(fmt.Errorf("error handling connection: %v", err))
						}
					}()
				}
			}
		}
	}()
	return func() { listener.Close() }, nil

}

func (r *roundRobin) handle(conn net.Conn) error {
	klog.V(6).Info("Receiving connection")
	transport, upgrader, err := spdy.RoundTripperFor(r.restConfig)
	if err != nil {
		return err
	}
	nativeClient, err := kubernetes.NewForConfig(r.restConfig)
	if err != nil {
		return err
	}
	podList, err := nativeClient.CoreV1().
		Pods(r.proxyServerNamespace).
		List(context.TODO(), metav1.ListOptions{
			LabelSelector: r.podSelector,
		})
	if err != nil {
		return err
	}

	candidates := make([]*corev1.Pod, 0)
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		pod := pod
		candidates = append(candidates, &pod)
	}
	if len(candidates) == 0 {
		return errors.New("no valid proxy server pod instances found")
	}

	r.lock.Lock()
	currentId := r.reqId
	r.reqId++
	r.lock.Unlock()

	podIdx := rand.Intn(len(candidates))
	pod := candidates[podIdx]
	klog.Infof("Forwarding to pod %v", pod.Name)
	req := nativeClient.RESTClient().
		Post().
		Prefix("api", "v1").
		Resource("pods").
		Namespace(r.proxyServerNamespace).
		Name(pod.Name).
		SubResource("portforward")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	streamConn, _, err := dialer.Dial(PortForwardProtocolV1Name)
	if err != nil {
		return err
	}
	defer streamConn.Close()

	// create error stream
	headers := http.Header{}
	headers.Set(corev1.StreamType, corev1.StreamTypeError)
	headers.Set(corev1.PortHeader, fmt.Sprintf("%d", r.targetPort))
	headers.Set(corev1.PortForwardRequestIDHeader, strconv.Itoa(currentId))

	errorStream, err := streamConn.CreateStream(headers)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error creating error stream for port %d -> %d: %v", r.targetPort, r.targetPort, err))
		return err
	}
	// we're not writing to this stream
	errorStream.Close()

	errorChan := make(chan error)
	go func() {
		message, err := io.ReadAll(errorStream)
		switch {
		case err != nil:
			errorChan <- fmt.Errorf("error reading from error stream for port %d -> %d: %v", r.targetPort, r.targetPort, err)
		case len(message) > 0:
			errorChan <- fmt.Errorf("an error occurred forwarding %d -> %d: %v", r.targetPort, r.targetPort, string(message))
		}
		close(errorChan)
	}()

	// create data stream
	headers.Set(corev1.StreamType, corev1.StreamTypeData)
	dataStream, err := streamConn.CreateStream(headers)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error creating forwarding stream for port %d -> %d: %v", r.targetPort, r.targetPort, err))
		return err
	}

	localError := make(chan struct{})
	remoteDone := make(chan struct{})

	go func() {
		// Copy from the remote side to the local port.
		if _, err := io.Copy(conn, dataStream); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			runtime.HandleError(fmt.Errorf("error copying from remote stream to local connection: %v", err))
		}

		// inform the select below that the remote copy is done
		close(remoteDone)
	}()

	go func() {
		// inform server we're not sending any more data after copy unblocks
		defer dataStream.Close()

		// Copy from the local port to the remote side.
		if _, err := io.Copy(dataStream, conn); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			runtime.HandleError(fmt.Errorf("error copying from local connection to remote stream: %v", err))
			// break out of the select below without waiting for the other copy to finish
			close(localError)
		}
	}()

	// wait for either a local->remote error or for copying from remote->local to finish
	defer func() {
		if err := conn.Close(); err != nil {
			runtime.HandleError(fmt.Errorf("failed closing port-forwarding connection: %v", err))
		}
	}()

	r.firstConnReceived.Store(true)

	for {
		select {
		case <-remoteDone:
			klog.V(6).Info("Connection closed from remote")
			return nil
		case <-localError:
			klog.V(6).Info("Connection closed due to local error")
			return nil
		case err := <-errorChan:
			if err != nil {
				runtime.HandleError(err)
			}
			klog.Infof("Connection closed due to error stream closed")
			return nil
		}
	}
}

var _ net.Conn = &conn{}

type conn struct {
	dataStream httpstream.Stream
}

func (c conn) Read(b []byte) (n int, err error) {
	return c.dataStream.Read(b)
}

func (c conn) Write(b []byte) (n int, err error) {
	return c.dataStream.Write(b)
}

func (c conn) Close() error {
	return c.dataStream.Close()
}

func (c conn) LocalAddr() net.Addr {
	return nil
}

func (c conn) RemoteAddr() net.Addr {
	return nil
}

func (c conn) SetDeadline(t time.Time) error {
	return errors.New("not implemented")
}

func (c conn) SetReadDeadline(t time.Time) error {
	return errors.New("not implemented")
}

func (c conn) SetWriteDeadline(t time.Time) error {
	return errors.New("not implemented")
}
