// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	"open-cluster-management.io/clusteradm/pkg/helpers/apply"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/util"
	version "open-cluster-management.io/clusteradm/pkg/helpers/version"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun, "force", o.force, "output-file", o.outputFile)
	o.values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
			Registry:    o.registry,		
			
		},
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)	

	if err != nil {
		klog.Errorf("unable to retrive version ", err)
		return err
	}
	
	o.values.BundleVersion = BundleVersion{
		RegistrationImageVersion:  versionBundle.Registration,
		PlacementImageVersion: versionBundle.Placement,	
		WorkImageVersion: versionBundle.Work,	
		OperatorImageVersion: versionBundle.Operator,
	}


	return nil
}

func (o *Options) validate() error {
	if o.force {
		return nil
	}
	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return err
	}
	apiExtensionsClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	installed, err := helpers.IsClusterManagerInstalled(apiExtensionsClient)
	if err != nil {
		return err
	}
	if installed {
		return fmt.Errorf("hub already initialized")
	}
	if len(o.registry) == 0 {
		return fmt.Errorf("registry should not be empty")
	}

	return nil
}

func (o *Options) run() error {
	token := fmt.Sprintf("%s.%s", o.values.Hub.TokenID, o.values.Hub.TokenSecret)
	output := make([]string, 0)
	reader := scenario.GetScenarioResourcesReader()

	kubeClient, apiExtensionsClient, dynamicClient, err := helpers.GetClients(o.ClusteradmFlags.KubectlFactory)
	if err != nil {
		return err
	}

	applierBuilder := &apply.ApplierBuilder{}
	applier := applierBuilder.WithClient(kubeClient, apiExtensionsClient, dynamicClient).Build()

	files := []string{
		"init/namespace.yaml",
	}
	if o.useBootstrapToken {
		files = append(files,
			"init/bootstrap-token-secret.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_cluster_role_binding.yaml",
		)
	} else {
		files = append(files,
			"init/bootstrap_sa.yaml",
			"init/bootstrap_cluster_role.yaml",
			"init/bootstrap_sa_cluster_role_binding.yaml",
		)
	}

	files = append(files,
		"init/clustermanager_cluster_role.yaml",
		"init/clustermanager_cluster_role_binding.yaml",
		"init/clustermanagers.crd.yaml",
		"init/clustermanager_sa.yaml",
	)

	out, err := applier.ApplyDirectly(reader, o.values, o.ClusteradmFlags.DryRun, "", files...)
	if err != nil {
		return err
	}
	output = append(output, out...)

	//if service-account wait for the sa secret
	if !o.useBootstrapToken && !o.ClusteradmFlags.DryRun {
		if err := waitForServiceAccountToken(kubeClient); err != nil {
			return err
		}
		token, err = helpers.GetBootstrapTokenFromSA(kubeClient)
		if err != nil {
			return err
		}
	}

	out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := waitUntilCRDReady(apiExtensionsClient); err != nil {
			return err
		}
	}
	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := waitUntilRegistrationOperatorReady(
			o.ClusteradmFlags.KubectlFactory,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}
	}

	out, err = applier.ApplyCustomResources(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/clustermanager.cr.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := waitUntilClusterManagerRegistrationReady(
			o.ClusteradmFlags.KubectlFactory,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}
	}

	restConfig, err := o.ClusteradmFlags.KubectlFactory.ToRESTConfig()
	if err != nil {
		return nil
	}

	cmd := fmt.Sprintf("%s join --hub-token %s --hub-apiserver %s",
		helpers.GetExampleHeader(),
		token,
		restConfig.Host)

	// if the init command prescribes a foreground installation, adds the `--wait`
	// flag to the join command to cohere the behavior of init and join commands.
	if o.wait {
		cmd = cmd + " --wait"
	}

	if len(o.outputJoinCommandFile) > 0 {
		sh, err := os.OpenFile(o.outputJoinCommandFile, os.O_CREATE|os.O_WRONLY, 0755)
		_, err = fmt.Fprintf(sh, "%s --cluster-name $1", cmd)
		if err != nil {
			return err
		}
		if err := sh.Close(); err != nil {
			return err
		}
	}

	fmt.Printf("The multicluster hub control plane has been initialized successfully!\n\n"+
		"You can now register cluster(s) to the hub control plane. Log onto those cluster(s) and run the following command:\n\n"+
		"    %s --cluster-name <cluster_name>\n\n"+
		"Replace <cluster_name> with a cluster name of your choice. For example, cluster1.\n\n",
		cmd,
	)

	return apply.WriteOutput(o.outputFile, output)
}

func waitForServiceAccountToken(kubeClient kubernetes.Interface) error {
	tokenSpinner := helpers.NewSpinner("Waiting for service account token...", time.Second)
	tokenSpinner.FinalMSG = "Service account token successfully signed.\n"
	tokenSpinner.Start()
	defer tokenSpinner.Stop()
	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
		return pollServiceAccountToken(kubeClient)
	})
}

func waitUntilCRDReady(apiExtensionsClient apiextensionsclient.Interface) error {
	b := retry.DefaultBackoff
	b.Duration = 200 * time.Millisecond

	crdSpinner := helpers.NewSpinner("Waiting for CRD to be ready...", time.Second)
	crdSpinner.FinalMSG = "CRD successfully registered.\n"
	crdSpinner.Start()
	defer crdSpinner.Stop()
	return helpers.WaitCRDToBeReady(
		apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", b)
}

func waitUntilRegistrationOperatorReady(f util.Factory, timeout int64) error {
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
	operatorSpinner := helpers.NewSpinnerWithStatus(
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
			phase.Store(helpers.GetSpinnerPodStatus(pod))
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

func waitUntilClusterManagerRegistrationReady(f util.Factory, timeout int64) error {
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
	clusterManagerSpinner := helpers.NewSpinnerWithStatus(
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
			phase.Store(helpers.GetSpinnerPodStatus(pod))
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

func pollServiceAccountToken(kubeClient kubernetes.Interface) (bool, error) {
	_, err := helpers.GetBootstrapTokenFromSA(kubeClient)
	switch {
	case errors.IsNotFound(err):
		return false, err
	case err != nil:
		return false, err
	}
	return true, nil
}
