// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"context"
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
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("timed out waiting for registration operator to become ready")
		}
		return err
	}
	return nil
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
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("timed out waiting for cluster manager registration to become ready")
		}
		return err
	}
	return nil
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
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("timed out waiting for multicluster controlplane to become ready")
		}
		return err
	}
	return nil
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

// IsFatalAPIError reports whether err is an API error that will not resolve by retrying.
func IsFatalAPIError(err error) bool {
	return k8serrors.IsForbidden(err) ||
		k8serrors.IsUnauthorized(err) ||
		k8serrors.IsInvalid(err) ||
		k8serrors.IsBadRequest(err) ||
		k8serrors.IsMethodNotSupported(err)
}

func waitUntilPodsReady(ctx context.Context, client kubernetes.Interface, namespace, labelSelector string, timeout int64, phase *atomic.Value) error {
	return wait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			if IsFatalAPIError(err) {
				return false, err
			}
			return false, nil
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			phase.Store(printer.GetSpinnerPodStatus(pod))
			if isPodReady(pod) {
				return true, nil
			}
		}
		return false, nil
	})
}

func isPodReady(pod *corev1.Pod) bool {
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
