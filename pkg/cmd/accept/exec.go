// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

const (
	groupNameBootstrap               = "system:bootstrappers:managedcluster"
	userNameSignatureBootstrapPrefix = "system:bootstrap:"
	userNameSignatureSA              = "system:serviceaccount:open-cluster-management:agent-registration-bootstrap"
	groupNameSA                      = "system:serviceaccounts:open-cluster-management"
	clusterLabel                     = "open-cluster-management.io/cluster-name"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.Values.Clusters = o.ClusterOptions.AllClusters().UnsortedList()
	klog.V(1).InfoS("accept options:", "dry-run", o.ClusteradmFlags.DryRun, "clusters", o.Values.Clusters, "wait", o.Wait)
	return nil
}

func (o *Options) Validate() error {
	if err := o.ClusteradmFlags.ValidateHub(); err != nil {
		return err
	}
	if err := o.ClusterOptions.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *Options) Run() error {
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
	var errs []error
	for _, clusterName := range o.Values.Clusters {
		if !o.Wait {
			approved, err := o.accept(kubeClient, clusterClient, clusterName, false)
			if err != nil {
				errs = append(errs, err)
			}
			if !approved {
				errs = append(errs, fmt.Errorf("no csr is approved yet for cluster %s", clusterName))
			}
		} else {
			err := wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, time.Duration(o.ClusteradmFlags.Timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
				approved, err := o.accept(kubeClient, clusterClient, clusterName, true)
				if !approved {
					return false, nil
				}
				if errors.IsNotFound(err) {
					return false, nil
				}
				return true, err
			})
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (o *Options) accept(kubeClient *kubernetes.Clientset, clusterClient *clusterclientset.Clientset, clusterName string, waitMode bool) (bool, error) {
	managedCluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
		clusterName,
		metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("fail to get managedcluster %s: %v", clusterName, err)
	}
	_, hasEksArn := managedCluster.Annotations["agent.open-cluster-management.io/managed-cluster-arn"]

	var approved bool
	if !hasEksArn {
		approved, err = o.approveCSR(kubeClient, clusterName, waitMode)
		if err != nil {
			return approved, fmt.Errorf("fail to approve the csr for cluster %s: %v", clusterName, err)
		}
	} else {
		approved = true
	}

	err = o.updateManagedCluster(clusterClient, clusterName)
	if err != nil {
		return approved, err
	}
	fmt.Fprintf(o.Streams.Out, "\n Your managed cluster %s has joined the Hub successfully. Visit https://open-cluster-management.io/scenarios or https://github.com/open-cluster-management-io/OCM/tree/main/solutions for next steps.\n", clusterName)
	return approved, nil
}

func (o *Options) approveCSR(kubeClient *kubernetes.Clientset, clusterName string, waitMode bool) (bool, error) {
	var hasApproved bool
	csrs, err := kubeClient.CertificatesV1().CertificateSigningRequests().List(context.TODO(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%v = %v", clusterLabel, clusterName),
		})
	if err != nil {
		return hasApproved, err
	}

	// Check if csr has the correct requester
	var passedCSRs []certificatesv1.CertificateSigningRequest
	csrRequesterMapper := map[string]string{}
	requesters := sets.New[string]()
	if o.SkipApproveCheck {
		passedCSRs = csrs.Items
	} else {
		for _, item := range csrs.Items {
			// Does not have the correct name prefix
			if !strings.HasPrefix(item.Spec.Username, userNameSignatureBootstrapPrefix) &&
				!strings.HasPrefix(item.Spec.Username, userNameSignatureSA) {
				continue
			}
			// Check groups
			groups := sets.NewString(item.Spec.Groups...)
			if !groups.Has(groupNameBootstrap) &&
				!groups.Has(groupNameSA) {
				continue
			}
			passedCSRs = append(passedCSRs, item)

			// parse the common name in the request
			cn, err := parseCSRCommonName(item.Spec.Request)
			if err != nil {
				fmt.Fprintf(o.Streams.ErrOut, "csr %s is not valid: %v", item.Name, err)
				continue
			}
			requesters.Insert(cn)
			csrRequesterMapper[item.Name] = cn
		}
	}

	// if there are multiple csr with different common name, it is possible that multiple agents is registered with the
	// same cluster name. We should stop here and let user specify a certain requester or enable skip-approve-check.
	requiredRequesters := sets.New[string](o.Requesters...)
	if len(requesters) > 1 {
		if requiredRequesters.Len() == 0 || !o.SkipApproveCheck {
			fmt.Fprintf(o.Streams.Out, "There are CSRs of different requesters: %s, approve is skipped "+
				"please specify the certain requesters with --requesters or set --skip-approve-check if "+
				"all CSRs need to be approved", strings.Join(requesters.UnsortedList(), ","))
			return false, nil
		}
	} else {
		// always approve if there is only one requester
		requiredRequesters = requiredRequesters.Union(requesters)
	}

	filteredRequesters := requesters.Intersection(requiredRequesters)

	// approve all csrs that are not approved.
	var csrToApprove []certificatesv1.CertificateSigningRequest
	for _, passedCSR := range passedCSRs {
		cn := csrRequesterMapper[passedCSR.Name]
		if !o.SkipApproveCheck && !filteredRequesters.Has(cn) {
			fmt.Fprintf(o.Streams.Out, "CSR %s with requester %s is not in the approve list\n", passedCSR.Name, cn)
			continue
		}
		// Check if already approved or denied
		approved, denied := GetCertApprovalCondition(&passedCSR.Status)
		// if already denied, then nothing to do
		if denied {
			fmt.Fprintf(o.Streams.Out, "CSR %s already denied\n", passedCSR.Name)
			continue
		}
		// if already approved, then nothing to do
		if approved {
			fmt.Fprintf(o.Streams.Out, "CSR %s already approved\n", passedCSR.Name)
			hasApproved = true
			continue
		}
		csrToApprove = append(csrToApprove, passedCSR)
	}

	// no csr found
	if len(csrToApprove) == 0 {
		if waitMode {
			fmt.Fprintf(o.Streams.Out, "no CSR to approve for cluster %s\n", clusterName)
		}

		return hasApproved, nil
	}
	// if dry-run don't approve
	if o.ClusteradmFlags.DryRun {
		return hasApproved, nil
	}

	var errs []error
	fmt.Fprintf(o.Streams.Out, "Starting approve csrs for the cluster %s\n", clusterName)
	for _, csr := range csrToApprove {
		if csr.Status.Conditions == nil {
			csr.Status.Conditions = make([]certificatesv1.CertificateSigningRequestCondition, 0)
		}
		csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
			Status:         corev1.ConditionTrue,
			Type:           certificatesv1.CertificateApproved,
			Reason:         fmt.Sprintf("%s Approve", helpers.GetExampleHeader()),
			Message:        fmt.Sprintf("This CSR was approved by %s certificate approve.", helpers.GetExampleHeader()),
			LastUpdateTime: metav1.Now(),
		})

		signingRequest := kubeClient.CertificatesV1().CertificateSigningRequests()
		if _, err := signingRequest.UpdateApproval(context.TODO(), csr.Name, &csr, metav1.UpdateOptions{}); err != nil {
			errs = append(errs, err)
		} else {
			fmt.Fprintf(o.Streams.Out, "CSR %s approved\n", csr.Name)
			hasApproved = true
		}
	}
	return hasApproved, utilerrors.NewAggregate(errs)
}

func (o *Options) updateManagedCluster(clusterClient *clusterclientset.Clientset, clusterName string) error {
	mc, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(),
		clusterName,
		metav1.GetOptions{})
	if err != nil {
		return err
	}
	if mc.Spec.HubAcceptsClient {
		fmt.Fprintf(o.Streams.Out, "hubAcceptsClient already set for managed cluster %s\n", clusterName)
		return nil
	}
	if o.ClusteradmFlags.DryRun {
		return nil
	}
	if !mc.Spec.HubAcceptsClient {
		patch := `{"spec":{"hubAcceptsClient":true}}`
		_, err = clusterClient.ClusterV1().ManagedClusters().Patch(context.TODO(), mc.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		if err != nil {
			return err
		}
		fmt.Fprintf(o.Streams.Out, "set hubAcceptsClient to true for managed cluster %s\n", clusterName)
	}
	return nil
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

func parseCSRCommonName(csr []byte) (string, error) {
	block, _ := pem.Decode(csr)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return "", fmt.Errorf("CSR was not recognized: PEM block type is not CERTIFICATE REQUEST")
	}

	x509cr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("CSR was not recognized: %v", err)
	}

	return x509cr.Subject.CommonName, nil
}
