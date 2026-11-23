// Copyright Contributors to the Open Cluster Management project
package accept

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubefake "k8s.io/client-go/kubernetes/fake"

	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
)

func newTestOptions() *Options {
	return &Options{
		ClusteradmFlags: &genericclioptionsclusteradm.ClusteradmFlags{},
		Streams:         genericiooptions.NewTestIOStreamsDiscard(),
	}
}

func TestAcceptAlreadyAcceptedNoRemainingCSR(t *testing.T) {
	managedCluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1"},
		Spec:       clusterv1.ManagedClusterSpec{HubAcceptsClient: true},
	}
	kubeClient := kubefake.NewSimpleClientset()
	clusterClient := clusterfake.NewSimpleClientset(managedCluster)

	o := newTestOptions()
	approved, err := o.accept(kubeClient, clusterClient, "cluster1", true)
	if err != nil {
		t.Fatalf("accept() returned unexpected error: %v", err)
	}
	if !approved {
		t.Errorf("expected accept() to report approved=true for an already accepted cluster with no remaining CSR, got false")
	}
}

func TestAcceptNotYetAcceptedNoCSR(t *testing.T) {
	managedCluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1"},
		Spec:       clusterv1.ManagedClusterSpec{HubAcceptsClient: false},
	}
	kubeClient := kubefake.NewSimpleClientset()
	clusterClient := clusterfake.NewSimpleClientset(managedCluster)

	o := newTestOptions()
	approved, err := o.accept(kubeClient, clusterClient, "cluster1", true)
	if err != nil {
		t.Fatalf("accept() returned unexpected error: %v", err)
	}
	if approved {
		t.Errorf("expected accept() to report approved=false when the cluster was never accepted and no CSR exists, got true")
	}
}

func TestRunWithClientStopsWaitingForAlreadyAcceptedCluster(t *testing.T) {
	managedCluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1"},
		Spec:       clusterv1.ManagedClusterSpec{HubAcceptsClient: true},
	}
	kubeClient := kubefake.NewSimpleClientset()
	clusterClient := clusterfake.NewSimpleClientset(managedCluster)

	o := newTestOptions()
	o.ClusteradmFlags.Timeout = 5
	o.Wait = true
	o.Values.Clusters = []string{"cluster1"}

	if err := o.runWithClient(kubeClient, clusterClient); err != nil {
		t.Errorf("runWithClient() returned an error for an already accepted cluster, want the wait to succeed immediately: %v", err)
	}
}
