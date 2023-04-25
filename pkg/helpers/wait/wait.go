// Copyright Contributors to the Open Cluster Management project
package wait

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/cmd/util"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/printer"
)

func WaitUntilCRDReady(apiExtensionsClient apiextensionsclient.Interface, crdName string, wait bool) error {
	b := retry.DefaultBackoff
	b.Duration = 200 * time.Millisecond

	if wait {
		crdSpinner := printer.NewSpinner("Waiting for CRD to be ready...", time.Second)
		crdSpinner.FinalMSG = "CRD successfully registered.\n"
		crdSpinner.Start()
		defer crdSpinner.Stop()
	}
	return helpers.WaitCRDToBeReady(apiExtensionsClient, crdName, b, wait)
}

func WaitUntilRegistrationOperatorReady(f util.Factory, timeout int64) error {
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
		text,
		time.Second,
		"Registration operator is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	operatorSpinner.Start()
	defer operatorSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods("open-cluster-management").
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					LabelSelector:  "app=cluster-manager",
				})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
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
		})
}

func WaitUntilClusterManagerRegistrationReady(f util.Factory, timeout int64) error {
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
		text,
		time.Second,
		"ClusterManager registration is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	clusterManagerSpinner.Start()
	defer clusterManagerSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods("open-cluster-management-hub").
				Watch(context.TODO(), metav1.ListOptions{
					TimeoutSeconds: &timeout,
					LabelSelector:  "app=clustermanager-registration-controller",
				})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
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
		})
}

func WaitUntilMulticlusterControlplaneReady(f util.Factory, ns string, timeout int64) error {
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
		text,
		time.Second,
		"Multicluster controlplane is now available.\n",
		func() string {
			return phase.Load().(string)
		})
	clusterManagerSpinner.Start()
	defer clusterManagerSpinner.Stop()

	return helpers.WatchUntil(
		func() (watch.Interface, error) {
			return client.CoreV1().Pods(ns).Watch(context.TODO(), metav1.ListOptions{
				TimeoutSeconds: &timeout,
				LabelSelector:  "app=multicluster-controlplane",
			})
		},
		func(event watch.Event) bool {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return false
			}
			phase.Store(printer.GetSpinnerPodStatus(pod))
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
		})
}

func WaitUntilMulticlusterControlplaneKubeconfigReady(f util.Factory, ns string, b wait.Backoff) error {
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
		_, err := client.CoreV1().Secrets(ns).Get(context.TODO(), "multicluster-controlplane-kubeconfig", metav1.GetOptions{})
		return err
	})
	return errGet
}
