// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/cmd/util"

	"open-cluster-management.io/clusteradm/pkg/config"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

//nolint:revive
func WaitUntilCRDReady(ctx context.Context, w io.Writer, apiExtensionsClient apiextensionsclient.Interface, crdName string, wait bool) error {
	b := retry.DefaultBackoff
	b.Duration = 200 * time.Millisecond

	if wait {
		crdSpinner := printer.NewSpinner(w, "Waiting for CRD to be ready...", time.Second)
		crdSpinner.FinalMSG = "CRD successfully registered.\n"
		crdSpinner.Start()
		defer crdSpinner.Stop()
	}
	return helpers.WaitCRDToBeReady(ctx, apiExtensionsClient, crdName, b, wait)
}

//nolint:revive
func WaitUntilRegistrationOperatorReady(ctx context.Context, w io.Writer, f util.Factory, timeout int64, appLabel string) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	text := "Waiting for registration operator to become ready..."
	operatorSpinner := printer.NewSpinnerWithStatus(
		w,
		text,
		time.Second,
		"Registration operator is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	operatorSpinner.Start()
	defer operatorSpinner.Stop()

	err = waitUntilPodsReady(ctx, client, "open-cluster-management",
		fmt.Sprintf("%v=%v", config.LabelApp, appLabel), timeout, phase)
	return TimeoutError(err, phase, "timed out waiting for registration operator to become ready")
}

//nolint:revive
func WaitUntilClusterManagerRegistrationReady(ctx context.Context, w io.Writer, f util.Factory, timeout int64) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	text := "Waiting for cluster manager registration to become ready..."
	clusterManagerSpinner := printer.NewSpinnerWithStatus(
		w,
		text,
		time.Second,
		"ClusterManager registration is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	clusterManagerSpinner.Start()
	defer clusterManagerSpinner.Stop()

	err = waitUntilPodsReady(ctx, client, "open-cluster-management-hub",
		"app=clustermanager-registration-controller", timeout, phase)
	return TimeoutError(err, phase, "timed out waiting for cluster manager registration to become ready")
}

//nolint:revive
func WaitUntilMulticlusterControlplaneReady(ctx context.Context, w io.Writer, f util.Factory, ns string, timeout int64) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	phase := &atomic.Value{}
	phase.Store("")
	text := "Waiting for multicluster controlplane to become ready..."
	clusterManagerSpinner := printer.NewSpinnerWithStatus(
		w,
		text,
		time.Second,
		"Multicluster controlplane is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	clusterManagerSpinner.Start()
	defer clusterManagerSpinner.Stop()

	err = waitUntilPodsReady(ctx, client, ns, "app=multicluster-controlplane", timeout, phase)
	return TimeoutError(err, phase, "timed out waiting for multicluster controlplane to become ready")
}

//nolint:revive
func WaitUntilMulticlusterControlplaneKubeconfigReady(ctx context.Context, f util.Factory, ns string, b wait.Backoff) error {
	var restConfig *rest.Config
	restConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	errGet := retry.OnError(b, func(err error) bool {
		if err != nil {
			fmt.Printf("wait for kubeconfig to be ready\n")
			return true
		}
		return false
	}, func() error {
		_, err := client.CoreV1().Secrets(ns).Get(ctx, "multicluster-controlplane-kubeconfig", metav1.GetOptions{})
		return err
	})
	return errGet
}

// IsFatalAPIError reports whether err will not resolve by retrying.
func IsFatalAPIError(err error) bool {
	if err == nil {
		return false
	}
	return k8serrors.IsForbidden(err) ||
		k8serrors.IsUnauthorized(err) ||
		k8serrors.IsInvalid(err) ||
		k8serrors.IsBadRequest(err) ||
		k8serrors.IsMethodNotSupported(err) ||
		isFatalTransportError(err)
}

// HandlePollError classifies err for a wait.PollUntilContextTimeout condition.
// Fatal errors abort the wait. Retryable errors are stored in status so spinners
// can show why polling is still running. Context cancellation and deadline are
// left to the poller via ctx.Done().
func HandlePollError(ctx context.Context, err error, status *atomic.Value) (bool, error) {
	handleStatus := func(statusMsg string, err error) (bool, error) {
		if status != nil {
			status.Store(statusMsg)
		}
		return false, err
	}

	if err == nil {
		return handleStatus("", nil)
	}
	if ctx.Err() != nil &&
		(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return false, nil
	}
	if IsFatalAPIError(err) {
		return handleStatus(err.Error(), err)
	}
	return handleStatus(err.Error(), nil)
}

// TimeoutError returns a timeout message that includes the last spinner status
// when err is a deadline. Other errors are returned unchanged. A nil err returns nil.
func TimeoutError(err error, status *atomic.Value, format string, a ...any) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	msg := fmt.Sprintf(format, a...)
	if status != nil {
		if s, ok := status.Load().(string); ok && s != "" {
			return fmt.Errorf("%s (%s): %w", msg, s, err)
		}
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func isFatalTransportError(err error) bool {
	var (
		unknownAuthority x509.UnknownAuthorityError
		hostname         x509.HostnameError
		certInvalid      x509.CertificateInvalidError
		systemRoots      x509.SystemRootsError
		certVerify       *tls.CertificateVerificationError
	)
	return errors.As(err, &unknownAuthority) ||
		errors.As(err, &hostname) ||
		errors.As(err, &certInvalid) ||
		errors.As(err, &systemRoots) ||
		errors.As(err, &certVerify)
}

func waitUntilPodsReady(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string, timeout int64, phase *atomic.Value) error {
	return wait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return HandlePollError(ctx, err, phase)
		}
		allPodsReady := true
		podCount := 0
		for _, pod := range pods.Items {
			// Ignore terminating, Failed, and Succeeded pods
			if pod.DeletionTimestamp != nil ||
				pod.Status.Phase == corev1.PodFailed ||
				pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			podCount++
			if !isPodReady(pod) {
				allPodsReady = false
				phase.Store(printer.GetSpinnerPodStatus(pod))
			} else if allPodsReady {
				phase.Store(printer.GetSpinnerPodStatus(pod))
			}
		}
		if podCount == 0 {
			return HandlePollError(ctx, fmt.Errorf("no pods found"), phase)
		}
		return allPodsReady, nil
	})
}

func isPodReady(pod corev1.Pod) bool {
	conds := make([]metav1.Condition, len(pod.Status.Conditions))
	for i := range pod.Status.Conditions {
		conds[i] = metav1.Condition{
			Type:    string(pod.Status.Conditions[i].Type),
			Status:  metav1.ConditionStatus(pod.Status.Conditions[i].Status),
			Reason:  pod.Status.Conditions[i].Reason,
			Message: pod.Status.Conditions[i].Message,
		}
	}
	return meta.IsStatusConditionTrue(conds, "Ready")
}
