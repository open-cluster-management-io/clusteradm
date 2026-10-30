// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/client/cluster/clientset/versioned/typed/cluster/v1"
	"open-cluster-management.io/clusteradm/pkg/helpers"
)

const (
	groupNameBootstrap               = "system:bootstrappers:managedcluster"
	userNameSignatureBootstrapPrefix = "system:bootstrap:"
	userNameSignatureSA              = "system:serviceaccount:open-cluster-management:agent-registration-bootstrap"
	userNameGRPCSignatureSA          = "system:serviceaccount:open-cluster-management-hub:grpc-server-sa"
	groupNameSA                      = "system:serviceaccounts:open-cluster-management"
	groupNameGRPC                    = "system:serviceaccounts:open-cluster-management-hub"
	clusterLabel                     = "open-cluster-management.io/cluster-name"
)

// errDeniedRegistrationCSR is returned when a denied registration CSR is the only
// blocker. Polling cannot approve it.
var errDeniedRegistrationCSR = errors.New("denied registration CSR")

func (o *Options) complete(_ *cobra.Command, _ []string) (err error) {
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

func (o *Options) Run(ctx context.Context) error {
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
	return o.runWithClient(ctx, kubeClient, clusterClient.ClusterV1().ManagedClusters())
}

func (o *Options) runWithClient(ctx context.Context, kubeClient kubernetes.Interface, clusterClient clusterv1.ManagedClusterInterface) error {
	var errs []error
	for _, clusterName := range o.Values.Clusters {
		if !o.Wait {
			approved, err := o.accept(ctx, kubeClient, clusterClient, clusterName, false)
			if err != nil {
				errs = append(errs, err)
			}
			// A denied CSR already explains why accept stopped. The generic message is for "not yet".
			if !approved && !errors.Is(err, errDeniedRegistrationCSR) {
				errs = append(errs, fmt.Errorf("no csr is approved yet for cluster %s", clusterName))
			}
		} else {
			err := wait.PollUntilContextTimeout(ctx, 3*time.Second, time.Duration(o.ClusteradmFlags.Timeout)*time.Second, true, func(ctx context.Context) (bool, error) {
				approved, err := o.accept(ctx, kubeClient, clusterClient, clusterName, true)
				if errors.Is(err, errDeniedRegistrationCSR) {
					return false, err
				}
				if !approved {
					return false, nil
				}
				if apierrors.IsNotFound(err) {
					return false, nil
				}
				return true, err
			})
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (o *Options) accept(ctx context.Context, kubeClient kubernetes.Interface, clusterClient clusterv1.ManagedClusterInterface, clusterName string, waitMode bool) (bool, error) {
	managedCluster, err := clusterClient.Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("fail to get managedcluster %s: %v", clusterName, err)
	}

	var approved bool

	// when a managed cluster registers with hub using awsirsa registration-auth, it will add this annotation
	// to ManagedCluster resource, presence of which is used to decide the requested authentication type.
	// awrirsa authentication doesn't create CSR on hub, hence there is nothing to approve
	_, hasEksArn := managedCluster.Annotations["agent.open-cluster-management.io/managed-cluster-arn"]
	if hasEksArn {
		approved = true
	} else {
		csrApproved, csrPending, err := o.approveCSR(ctx, kubeClient, clusterName, waitMode)
		if err != nil {
			if errors.Is(err, errDeniedRegistrationCSR) {
				return false, err
			}
			return csrApproved, fmt.Errorf("fail to approve the csr for cluster %s: %v", clusterName, err)
		}

		approved = csrApproved || (managedCluster.Spec.HubAcceptsClient && !csrPending)
	}

	// Set hubAcceptsClient only after approval. Writing it earlier makes the next
	// poll look like an already-accepted cluster when no CSR exists yet.
	if !approved {
		return false, nil
	}

	if err := o.updateManagedCluster(ctx, clusterClient, clusterName); err != nil {
		return approved, err
	}

	fmt.Fprintf(o.Streams.Out, "\n Your managed cluster %s has joined the Hub successfully. Visit https://open-cluster-management.io/docs/scenarios or https://github.com/open-cluster-management-io/OCM/tree/main/solutions for next steps.\n", clusterName)

	return approved, nil
}

// approveCSR approves registration CSRs for clusterName.
// It reports whether a CSR is approved, and whether an unapproved registration CSR is still present.
func (o *Options) approveCSR(ctx context.Context, kubeClient kubernetes.Interface, clusterName string, waitMode bool) (bool, bool, error) {
	var hasApproved bool
	csrs, err := kubeClient.CertificatesV1().CertificateSigningRequests().List(ctx,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%v = %v", clusterLabel, clusterName),
		})
	if err != nil {
		return hasApproved, false, err
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
				!strings.HasPrefix(item.Spec.Username, userNameSignatureSA) &&
				!strings.HasPrefix(item.Spec.Username, userNameGRPCSignatureSA) {
				continue
			}
			// Check groups
			groups := sets.NewString(item.Spec.Groups...)
			if !groups.Has(groupNameBootstrap) &&
				!groups.Has(groupNameSA) &&
				!groups.Has(groupNameGRPC) {
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

	// Approved CSR names. Objects approved by this call still have their previous status stored.
	approvedNow := sets.New[string]()

	// if there are multiple csr with different common name, it is possible that multiple agents is registered with the
	// same cluster name. We should stop here and let user specify a certain requester or enable skip-approve-check.
	requiredRequesters := sets.New(o.Requesters...)
	if len(requesters) > 1 {
		if requiredRequesters.Len() == 0 || !o.SkipApproveCheck {
			pending, denied := registrationCSRBlockers(passedCSRs, approvedNow)
			if pending {
				fmt.Fprintf(o.Streams.Out, "There are CSRs of different requesters: %s, approve is skipped "+
					"please specify the certain requesters with --requesters or set --skip-approve-check if "+
					"all CSRs need to be approved", strings.Join(requesters.UnsortedList(), ","))
				return false, true, nil
			}
			// Nothing left to approve. Report the existing CSR states without the skip warning.
			return o.reportSettledCSRs(clusterName, passedCSRs, denied)
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
			approvedNow.Insert(passedCSR.Name)
			continue
		}
		csrToApprove = append(csrToApprove, passedCSR)
	}

	// no csr found
	if len(csrToApprove) == 0 {
		approvedResult, blocked, err := o.approvalResult(clusterName, hasApproved, passedCSRs, approvedNow, nil)
		if err == nil && waitMode {
			fmt.Fprintf(o.Streams.Out, "no CSR to approve for cluster %s\n", clusterName)
		}
		return approvedResult, blocked, err
	}
	// if dry-run don't approve
	if o.ClusteradmFlags.DryRun {
		return o.approvalResult(clusterName, hasApproved, passedCSRs, approvedNow, nil)
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
		if _, err := signingRequest.UpdateApproval(ctx, csr.Name, &csr, metav1.UpdateOptions{}); err != nil {
			errs = append(errs, err)
		} else {
			fmt.Fprintf(o.Streams.Out, "CSR %s approved\n", csr.Name)
			hasApproved = true
			approvedNow.Insert(csr.Name)
		}
	}
	return o.approvalResult(clusterName, hasApproved, passedCSRs, approvedNow, utilerrors.NewAggregate(errs))
}

// registrationCSRBlockers reports registration CSRs that still block a completed join.
// pending is true for CSRs that are neither approved nor denied, including skipped and dry-run CSRs.
// denied lists denied CSRs. A CSR approved by this call is omitted; its stored status is still the previous one.
func registrationCSRBlockers(csrs []certificatesv1.CertificateSigningRequest, approvedNow sets.Set[string]) (pending bool, denied []string) {
	for _, csr := range csrs {
		if approvedNow.Has(csr.Name) {
			continue
		}
		approved, isDenied := GetCertApprovalCondition(&csr.Status)
		if approved && !isDenied {
			continue
		}
		if isDenied {
			denied = append(denied, csr.Name)
			continue
		}
		pending = true
	}
	return pending, denied
}

func (o *Options) approvalResult(clusterName string, hasApproved bool, csrs []certificatesv1.CertificateSigningRequest, approvedNow sets.Set[string], extraErr error) (bool, bool, error) {
	pending, denied := registrationCSRBlockers(csrs, approvedNow)
	blocked := pending || len(denied) > 0
	if extraErr != nil {
		return hasApproved, blocked, extraErr
	}
	if !hasApproved && !pending && len(denied) > 0 {
		return false, true, deniedRegistrationCSRError(clusterName, denied)
	}
	return hasApproved, blocked, nil
}

// reportSettledCSRs prints the state of CSRs that are already approved or denied.
// It returns approved when any registration CSR is approved, and a denied-CSR error when every one is denied.
func (o *Options) reportSettledCSRs(clusterName string, csrs []certificatesv1.CertificateSigningRequest, denied []string) (bool, bool, error) {
	hasApproved := false
	for _, csr := range csrs {
		approved, isDenied := GetCertApprovalCondition(&csr.Status)
		if isDenied {
			fmt.Fprintf(o.Streams.Out, "CSR %s already denied\n", csr.Name)
			continue
		}
		if approved {
			fmt.Fprintf(o.Streams.Out, "CSR %s already approved\n", csr.Name)
			hasApproved = true
		}
	}
	if len(denied) > 0 && !hasApproved {
		return false, true, deniedRegistrationCSRError(clusterName, denied)
	}
	return hasApproved, false, nil
}

func deniedRegistrationCSRError(clusterName string, names []string) error {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if len(sorted) == 1 {
		return fmt.Errorf("CSR %s for cluster %s is denied; delete it so the agent can request a new certificate: %w", sorted[0], clusterName, errDeniedRegistrationCSR)
	}
	return fmt.Errorf("CSRs %s for cluster %s are denied; delete them so the agent can request new certificates: %w", strings.Join(sorted, ", "), clusterName, errDeniedRegistrationCSR)
}

func (o *Options) updateManagedCluster(ctx context.Context, clusterClient clusterv1.ManagedClusterInterface, clusterName string) error {
	mc, err := clusterClient.Get(ctx, clusterName, metav1.GetOptions{})
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
		_, err = clusterClient.Patch(ctx, mc.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
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
