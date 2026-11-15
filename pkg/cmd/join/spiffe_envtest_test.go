// Copyright Contributors to the Open Cluster Management project
package join

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	operatorclient "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
)

func envJoinOptions(t *testing.T, cfg *rest.Config, auth string) (*Options, *cobra.Command, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	flags := envFlags(cfg)
	flags.Timeout = 60
	streams := genericiooptions.IOStreams{Out: out, ErrOut: out}
	cmd := NewCmd(flags, streams)
	o := newOptions(flags, streams)
	o.registrationAuth = auth
	o.clusterName = "cluster-" + rand.String(5)
	o.registry = "quay.io/open-cluster-management"
	o.bundleVersion = "default"
	o.mode = "default"
	o.createNameSpace = true
	o.resourceQosClass = "Default"
	o.addonKubeClientRegistrationAuth = "csr"
	o.spiffe = spiffeOptions{hubGRPCEndpoint: "ocm-grpc.hub.internal:443", hubSPIFFEID: testHubID}
	return o, cmd, out
}

func joinOnCluster(t *testing.T, o *Options, cmd *cobra.Command) error {
	t.Helper()
	if err := o.complete(cmd, nil); err != nil {
		return err
	}
	if err := o.validate(context.TODO()); err != nil {
		return err
	}
	return o.run(context.TODO())
}

// Without --wait, join applies the Klusterlet and says what to do once it is SPIFFEAdmissionReady.
func TestJoinSPIFFEAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	o, cmd, out := envJoinOptions(t, cfg, operatorv1.SPIFFEAuthType)
	o.outputFile = filepath.Join(t.TempDir(), "applied.yaml")
	if err := joinOnCluster(t, o, cmd); err != nil {
		t.Fatal(err)
	}
	k, err := operatorclient.NewForConfigOrDie(cfg).OperatorV1().Klusterlets().Get(context.TODO(), "klusterlet", metav1.GetOptions{})
	if err != nil || k.Spec.RegistrationConfiguration.RegistrationDriver.AuthType != operatorv1.SPIFFEAuthType {
		t.Fatalf("Klusterlet %+v, %v", k, err)
	}
	if applied, _ := os.ReadFile(o.outputFile); !strings.Contains(string(applied), "kind: Klusterlet") {
		t.Errorf("output file:\n%s", applied)
	}
	if !strings.Contains(out.String(), "SPIFFEAdmissionReady") {
		t.Errorf("output:\n%s", out)
	}
}

// With --wait, join returns once the operator has secured the namespaces, and prints the IDs.
func TestJoinSPIFFEWaitAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	kube := kubernetes.NewForConfigOrDie(cfg)
	operators := operatorclient.NewForConfigOrDie(cfg)
	// A running Klusterlet operator, so join doesn't deploy and wait for one.
	if _, err := kube.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: OperatorNamespace}}, metav1.CreateOptions{}); err != nil && !strings.Contains(err.Error(), "exists") {
		t.Fatal(err)
	}
	labels := map[string]string{"app": "klusterlet"}
	_, err := kube.AppsV1().Deployments(OperatorNamespace).Create(context.TODO(), &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: DefaultOperatorName, Namespace: OperatorNamespace},
		Spec: appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "operator", Image: "operator"}}}}}},
		metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "exists") {
		t.Fatal(err)
	}
	// It may exist already, from another join test: mark it available either way.
	d, err := kube.AppsV1().Deployments(OperatorNamespace).Get(context.TODO(), DefaultOperatorName, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	d.Status.Conditions = []appsv1.DeploymentCondition{{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue}}
	if _, err := kube.AppsV1().Deployments(OperatorNamespace).UpdateStatus(context.TODO(), d, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	o, cmd, out := envJoinOptions(t, cfg, operatorv1.SPIFFEAuthType)
	o.wait = true
	done := make(chan error)
	go func() { done <- joinOnCluster(t, o, cmd) }()

	// The operator's part: record the IDs and become SPIFFEAdmissionReady.
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		k, err := operators.OperatorV1().Klusterlets().Get(context.TODO(), "klusterlet", metav1.GetOptions{})
		if err == nil {
			k.Status.SPIFFE = &operatorv1.KlusterletSPIFFEStatus{AgentIDs: []operatorv1.SPIFFEAgentID{
				{Role: "registration", SPIFFEID: "spiffe://ocm-hub.example.org/ocm/cluster/" + o.clusterName + "/agent/registration"}}}
			meta.SetStatusCondition(&k.Status.Conditions, metav1.Condition{Type: operatorv1.ConditionSPIFFEAdmissionReady,
				Status: metav1.ConditionTrue, Reason: "Recorded"})
			if _, err := operators.OperatorV1().Klusterlets().UpdateStatus(context.TODO(), k, metav1.UpdateOptions{}); err == nil {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(90 * time.Second):
		t.Fatal("join didn't return")
	}
	if !strings.Contains(out.String(), "/agent/registration") {
		t.Errorf("output:\n%s", out)
	}
}

// A token join reads the hub's CA from cluster-info and checks the hub with the token.
func TestJoinWithTokenAgainstCluster(t *testing.T) {
	cfg := testEnvConfig(t)
	kube := kubernetes.NewForConfigOrDie(cfg)
	info := clientcmdapi.NewConfig()
	info.Clusters[""] = &clientcmdapi.Cluster{Server: cfg.Host, CertificateAuthorityData: cfg.CAData}
	data, err := clientcmd.Write(*info)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().ConfigMaps("kube-public").Create(context.TODO(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-info", Namespace: "kube-public"},
		Data:       map[string]string{"kubeconfig": string(data)}}, metav1.CreateOptions{}); err != nil &&
		!strings.Contains(err.Error(), "exists") {
		t.Fatal(err)
	}
	o, cmd, _ := envJoinOptions(t, cfg, "csr")
	o.token, o.hubAPIServer = testHubToken, cfg.Host
	o.addonKubeClientRegistrationAuth = "token"
	if err := o.complete(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := o.validate(context.TODO()); err != nil {
		t.Fatal(err)
	}
	if o.HubConfig == nil || o.HubConfig.Clusters[0].Cluster.Server != cfg.Host {
		t.Errorf("hub config %+v", o.HubConfig)
	}

	// spiffe add-on agents need spiffe Klusterlet agents.
	o.addonKubeClientRegistrationAuth = operatorv1.SPIFFEAuthType
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "needs registration-auth spiffe") {
		t.Errorf("spiffe add-ons without spiffe: %v", err)
	}
	o.addonKubeClientRegistrationAuth = "token"

	// A proxy CA that can't be read (last: it leaves the proxy in the hub kubeconfig).
	o.proxyURL, o.proxyCAFile = "http://proxy.example.com:3128", filepath.Join(t.TempDir(), "missing-ca.crt")
	if err := o.validate(context.TODO()); err == nil || !strings.Contains(err.Error(), "missing-ca.crt") {
		t.Errorf("unreadable proxy CA: %v", err)
	}
}

func TestJoinCompleteAndValidateErrors(t *testing.T) {
	o, cmd, _ := spiffeJoinOptions(t)
	o.registrationAuth = "csr"
	if err := o.complete(cmd, nil); err == nil || !strings.Contains(err.Error(), "token is missing") {
		t.Errorf("no token: %v", err)
	}
	o.token = "t"
	if err := o.complete(cmd, nil); err == nil || !strings.Contains(err.Error(), "hub-server is missing") {
		t.Errorf("no hub server: %v", err)
	}

}

// The hub can't be reached for its CA, and a cluster name that isn't a valid name fails preflight.
func TestJoinHubUnreachableAndBadClusterName(t *testing.T) {
	o, cmd, _ := spiffeJoinOptions(t)
	o.registrationAuth, o.token, o.hubAPIServer = "csr", "t", "https://127.0.0.1:1"
	if err := o.complete(cmd, nil); err == nil {
		t.Error("no error for an unreachable hub")
	}

	o, cmd, _ = spiffeJoinOptions(t)
	o.clusterName = "Bad_Name"
	if err := o.complete(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if err := o.validate(context.TODO()); err == nil {
		t.Error("no error for an invalid cluster name")
	}
}
