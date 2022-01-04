// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"open-cluster-management.io/clusteradm/pkg/helpers"

	"github.com/spf13/cobra"
	certificatesv1 "k8s.io/api/certificates/v1"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

const (
	groupNameBootstrap               = "system:bootstrappers:managedcluster"
	userNameSignatureBootstrapPrefix = "system:bootstrap:"
	userNameSignatureSA              = "system:serviceaccount:open-cluster-management:cluster-bootstrap"
	groupNameSA                      = "system:serviceaccounts:open-cluster-management"
	clusterLabel                     = "open-cluster-management.io/cluster-name"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("accept options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.clusters, "wait", o.wait)
	alreadyProvidedCluster := make(map[string]bool)
	clusters := make([]string, 0)
	if o.clusters != "" {
		cs := strings.Split(o.clusters, ",")
		for _, c := range cs {
			if _, ok := alreadyProvidedCluster[c]; !ok {
				alreadyProvidedCluster[c] = true
				clusters = append(clusters, strings.TrimSpace(c))
			}
		}
		o.values.clusters = clusters
	} else {
		return fmt.Errorf("values or name are missing")
	}
	klog.V(3).InfoS("values:", "clusters", o.values.clusters)
	return nil
}

func (o *Options) validate() error {
	return nil
}

func (o *Options) run() error {
	kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
	if err != nil {
		return err
	}
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	clusterClient, err := clusterclientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	return o.runWithClient(kubeClient, clusterClient)
}

func (o *Options) runWithClient(kubeClient *kubernetes.Clientset, clusterClient *clusterclientset.Clientset) (err error) {
	for _, clusterName := range o.values.clusters {
		if !o.wait {
			var csrApproved bool
			csrApproved, err = o.accept(kubeClient, clusterClient, clusterName, false)
			if err == nil && !csrApproved {
				err = fmt.Errorf("no CSR to approve for cluster %s", clusterName)
			}
		} else {
			err = wait.PollImmediate(1*time.Second, time.Duration(o.ClusteradmFlags.Timeout)*time.Second, func() (bool, error) {
				return o.accept(kubeClient, clusterClient, clusterName, true)
			})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Options) accept(kubeClient *kubernetes.Clientset, clusterClient *clusterclientset.Clientset, clusterName string, waitMode bool) (bool, error) {
	csrApproved, err := o.approveCSR(kubeClient, clusterName, waitMode)
	if err != nil {
		return false, err
	}
	mcUpdated, err := o.updateManagedCluster(clusterClient, clusterName)
	if err != nil {
		return false, err
	}
	if csrApproved && mcUpdated {
		return true, nil
	}
	return false, nil
}

func (o *Options) approveCSR(kubeClient *kubernetes.Clientset, clusterName string, waitMode bool) (bool, error) {
	csrs, err := kubeClient.CertificatesV1().CertificateSigningRequests().List(context.TODO(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%v = %v", clusterLabel, clusterName),
		})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	var csr *certificatesv1.CertificateSigningRequest
	for _, item := range csrs.Items {
		//Does not have the correct name prefix
		if !strings.HasPrefix(item.Spec.Username, userNameSignatureBootstrapPrefix) &&
			!strings.HasPrefix(item.Spec.Username, userNameSignatureSA) {
			continue
		}
		//Check groups
		groups := sets.NewString(item.Spec.Groups...)
		if !groups.Has(groupNameBootstrap) &&
			!groups.Has(groupNameSA) {
			continue
		}
		//Check if already approved or denied
		approved, denied := GetCertApprovalCondition(&item.Status)
		//if alreaady denied, then nothing to do
		if denied {
			fmt.Printf("CSR %s already denied\n", item.Name)
			return true, nil
		}
		//if alreaady approved, then nothing to do
		if approved {
			fmt.Printf("CSR %s already approved\n", item.Name)
			return true, nil
		}
		csr = &item
		break
	}

	//no csr found
	if csr == nil {
		if waitMode {
			fmt.Printf("no CSR to approve for cluster %s\n", clusterName)
		}
		return false, nil
	}
	//if dry-run don't approve
	if o.ClusteradmFlags.DryRun {
		return true, nil
	}
	if csr.Status.Conditions == nil {
		csr.Status.Conditions = make([]certificatesv1.CertificateSigningRequestCondition, 0)
	}

	csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Status:         corev1.ConditionTrue,
		Type:           certificatesv1.CertificateApproved,
		Reason:         fmt.Sprintf("%sApprove", helpers.GetExampleHeader()),
		Message:        fmt.Sprintf("This CSR was approved by %s certificate approve.", helpers.GetExampleHeader()),
		LastUpdateTime: metav1.Now(),
	})

	signingRequest := kubeClient.CertificatesV1().CertificateSigningRequests()
	if _, err := signingRequest.UpdateApproval(context.TODO(), csr.Name, csr, metav1.UpdateOptions{}); err != nil {
		return false, err
	}

	fmt.Printf("CSR %s approved\n", csr.Name)
	return true, nil
}

func (o *Options) updateManagedCluster(clusterClient *clusterclientset.Clientset, clusterName string) (bool, error) {
	mc, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
		clusterName,
		metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if mc.Spec.HubAcceptsClient {
		fmt.Printf("hubAcceptsClient already set for managed cluster %s\n", clusterName)
		return true, nil
	}
	if o.ClusteradmFlags.DryRun {
		return true, nil
	}
	if !mc.Spec.HubAcceptsClient {
		mc.Spec.HubAcceptsClient = true
		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), mc, metav1.UpdateOptions{})
		if err != nil {
			return false, err
		}
		fmt.Printf("set hubAcceptsClient to true for managed cluster %s\n", clusterName)
	}
	return true, nil
}

func GetCertApprovalCondition(status *certificatesv1.CertificateSigningRequestStatus) (approved bool, denied bool) {
	for _, c := range status.Conditions {
		if c.Type == certificatesv1.CertificateApproved {
			approved = true
		}
		if c.Type == certificatesv1.CertificateDenied {
			denied = true
		}
	}
	return
}
