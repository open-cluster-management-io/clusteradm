// Copyright Contributors to the Open Cluster Management project
package util

import (
	"context"
	"encoding/base64"
	"fmt"
	"open-cluster-management.io/clusteradm/pkg/version"
	"os"
	"time"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/clusteradm/pkg/config"
)

// WaitNamespaceDeleted receive a kubeconfigpath, a context name and a namespace name,
// then poll until the specific namespace is fully deleted or an error occurs.
func WaitNamespaceDeleted(restcfg *rest.Config, namespace string) error {
	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return wait.PollUntilContextCancel(context.TODO(), 1*time.Second, true, func(ctx context.Context) (bool, error) {
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		fmt.Printf("namespace %s still exists %v\n", ns.Name, ns.Status)
		return false, nil
	})
}

func DeleteClusterCSRs(restcfg *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	return clientset.CertificatesV1().CertificateSigningRequests().DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "open-cluster-management.io/cluster-name",
	})
}

func WaitClustersDeleted(restcfg *rest.Config) error {
	clientset, err := clusterclient.NewForConfig(restcfg)
	if err != nil {
		return err
	}

	gomega.Eventually(func() error {
		clusterList, err := clientset.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{})
		if errors.IsNotFound(err) || len(clusterList.Items) == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		for _, mcl := range clusterList.Items {
			err = clientset.ClusterV1().ManagedClusters().Delete(context.TODO(), mcl.Name, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
		return fmt.Errorf("not all clusters are deleted")
	}, time.Second*300, time.Second*2).Should(gomega.Succeed())

	return nil
}

// buildConfigFromFlags build rest config for specified context in the kubeconfigfile.
func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

func ValidateImagePullSecret(kubeClient kubernetes.Interface, expectedCred string, namespace string) error {
	pullSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), config.ImagePullSecret, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot find pull secret in %v ns. %v", namespace, err)
	}
	if base64.StdEncoding.EncodeToString(pullSecret.Data[".dockerconfigjson"]) != expectedCred {
		return fmt.Errorf("unexpected .dockerconfigjson %v of pull secret in ns %v.expected:%v",
			base64.StdEncoding.EncodeToString(pullSecret.Data[".dockerconfigjson"]), namespace, expectedCred)
	}

	return nil
}

func NewTestImagePullCredentialFile(fileName string) string {
	data := `{"auths":{}}`
	_ = os.WriteFile(fileName, []byte(data), 0600)
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func CleanupTestImagePullCredentialFile(fileName string) {
	_ = os.Remove(fileName)
}

func WaitClusterManagerApplied(operatorClient operatorclient.Interface, e2eConf *TestE2eConfig) {
	gomega.Expect(e2eConf).NotTo(gomega.BeNil())
	kubeClient, err := kubernetes.NewForConfig(e2eConf.Cluster().hub.kubeConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Eventually(func() error {
		cm, err := operatorClient.OperatorV1().ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if !meta.IsStatusConditionTrue(cm.Status.Conditions, operatorv1.ConditionClusterManagerApplied) {
			return fmt.Errorf("cluster manager is not applied")
		}

		con := meta.FindStatusCondition(cm.Status.Conditions, operatorv1.ConditionHubRegistrationDegraded)
		if con == nil {
			return fmt.Errorf("hub registration is not degraded")
		}
		if con.Status != metav1.ConditionFalse || con.Reason != operatorv1.ReasonRegistrationFunctional {
			return fmt.Errorf("hub registration is not functional")
		}

		deployments, err := kubeClient.AppsV1().Deployments(config.HubClusterNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, d := range deployments.Items {
			desiredReplicas := int32(1)
			if d.Spec.Replicas != nil {
				desiredReplicas = *(d.Spec.Replicas)
			}
			if desiredReplicas > d.Status.AvailableReplicas {
				return fmt.Errorf("deployment %v is available", d.Name)
			}
		}
		return nil

	}, time.Second*60, time.Second*2).Should(gomega.Succeed())
}

func CheckOperatorAndAgentVersion(mcl1KubeClient kubernetes.Interface, operatorBundleVersion, registrationBundleVersion string) error {
	operator, err := mcl1KubeClient.AppsV1().Deployments("open-cluster-management").Get(context.TODO(), "klusterlet", metav1.GetOptions{})
	if err != nil {
		return err
	}

	operatorVersion, err := version.GetVersionBundle(operatorBundleVersion, "")
	if err != nil {
		return err
	}

	if len(operator.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("klusterlet deployment has no containers")
	}
	if operator.Spec.Template.Spec.Containers[0].Image != fmt.Sprintf("quay.io/open-cluster-management/registration-operator:%s", operatorVersion.OCM) {
		return fmt.Errorf("version of the operator is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
	}

	registrationVersion, err := version.GetVersionBundle(registrationBundleVersion, "")
	if err != nil {
		return err
	}

	registration, err := mcl1KubeClient.AppsV1().Deployments("open-cluster-management-agent").Get(
		context.TODO(), "klusterlet-registration-agent", metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(registration.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("klusterlet-registration-agent deployment has no containers")
	}
	if registration.Spec.Template.Spec.Containers[0].Image != fmt.Sprintf("quay.io/open-cluster-management/registration:%s", registrationVersion.OCM) {
		return fmt.Errorf("version of the registration agent is not correct, get %s", registration.Spec.Template.Spec.Containers[0].Image)
	}

	return nil
}

func CheckOperatorAndManagerVersion(hubKubeClient kubernetes.Interface, operatorBundleVersion, registrationBundleVersion string) error {
	operator, err := hubKubeClient.AppsV1().Deployments("open-cluster-management").Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
	if err != nil {
		return err
	}

	operatorVersion, err := version.GetVersionBundle(operatorBundleVersion, "")
	if err != nil {
		return err
	}

	if len(operator.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("cluster manager deployment has no containers")
	}
	if operator.Spec.Template.Spec.Containers[0].Image != fmt.Sprintf("quay.io/open-cluster-management/registration-operator:%s", operatorVersion.OCM) {
		return fmt.Errorf("version of the operator is not correct, get %s", operator.Spec.Template.Spec.Containers[0].Image)
	}

	registrationVersion, err := version.GetVersionBundle(registrationBundleVersion, "")
	if err != nil {
		return err
	}

	registration, err := hubKubeClient.AppsV1().Deployments("open-cluster-management-hub").Get(
		context.TODO(), "cluster-manager-registration-controller", metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(registration.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("cluster-manager-registration-controller deployment has no containers")
	}
	if registration.Spec.Template.Spec.Containers[0].Image != fmt.Sprintf("quay.io/open-cluster-management/registration:%s", registrationVersion.OCM) {
		return fmt.Errorf("version of the registration controller is not correct, get %s", registration.Spec.Template.Spec.Containers[0].Image)
	}

	return nil
}
