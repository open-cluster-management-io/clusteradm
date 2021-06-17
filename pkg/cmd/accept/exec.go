// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

func (o *Options) runWithClient(kubeClient *kubernetes.Clientset, clusterClient *clusterclientset.Clientset) error {
	for _, clusterName := range o.values.clusters {

		csrs, err := kubeClient.CertificatesV1().CertificateSigningRequests().List(context.TODO(),
			metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%v = %v", clusterLabel, clusterName),
			})
		if err != nil {
			return err
		}
		var csr *certificatesv1.CertificateSigningRequest
		for _, item := range csrs.Items {
			//Does not have the correct name prefix
			if !strings.HasPrefix(item.Spec.Username, userNameSignatureBootstrapPrefix) &&
				item.Spec.Username != userNameSignatureSA {
				continue
			}
			//Check groups
			var group string
			for _, g := range item.Spec.Groups {
				if g == groupNameBootstrap || g == groupNameSA {
					group = g
					break
				}
			}
			//Does not contain the correct group
			if len(group) == 0 {
				continue
			}
			//Check if already approved or denied
			done := false
			for _, c := range item.Status.Conditions {
				if c.Type == certificatesv1.CertificateApproved || c.Type == certificatesv1.CertificateDenied {
					done = true
					break
				}
			}
			if done {
				continue
			}
			csr = &item
			break
		}

		if csr != nil {
			if !o.ClusteradmFlags.DryRun {
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

				kubeClient, err := o.ClusteradmFlags.KubectlFactory.KubernetesClientSet()
				if err != nil {
					return err
				}
				signingRequest := kubeClient.CertificatesV1().CertificateSigningRequests()
				if _, err := signingRequest.UpdateApproval(context.TODO(), csr.Name, csr, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			fmt.Printf("CSR %s approved\n", csr.Name)
		} else {
			fmt.Printf("no CSR to approve for cluster %s\n", clusterName)
		}

		mc, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
			clusterName,
			metav1.GetOptions{})
		if err != nil {
			return err
		}
		if !mc.Spec.HubAcceptsClient {
			if !o.ClusteradmFlags.DryRun {
				mc.Spec.HubAcceptsClient = true
				_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), mc, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			}
			fmt.Printf("set httpAcceptsClient to true for cluster %s\n", clusterName)
		} else {
			fmt.Printf("httpAcceptsClient already set for cluster %s\n", clusterName)
		}
	}
	return nil
}
