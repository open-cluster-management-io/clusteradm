// Copyright Contributors to the Open Cluster Management project
package init

import (
	"context"
	"fmt"
	ocmfeature "open-cluster-management.io/api/feature"
	genericclioptionsclusteradm "open-cluster-management.io/clusteradm/pkg/genericclioptions"
	"os"

	"github.com/spf13/cobra"
	"github.com/stolostron/applier/pkg/apply"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/preflight"
	"open-cluster-management.io/clusteradm/pkg/cmd/init/scenario"
	"open-cluster-management.io/clusteradm/pkg/helpers"
	clusteradmjson "open-cluster-management.io/clusteradm/pkg/helpers/json"
	preflightinterface "open-cluster-management.io/clusteradm/pkg/helpers/preflight"
	version "open-cluster-management.io/clusteradm/pkg/helpers/version"
	helperwait "open-cluster-management.io/clusteradm/pkg/helpers/wait"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	klog.V(1).InfoS("init options:", "dry-run", o.ClusteradmFlags.DryRun, "force", o.force, "output-file", o.outputFile)
	o.values = Values{
		Hub: Hub{
			TokenID:     helpers.RandStringRunes_az09(6),
			TokenSecret: helpers.RandStringRunes_az09(16),
			Registry:    o.registry,
		},
		RegistrationFeatures: genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubRegistrationFeatureGates),
		WorkFeatures:         genericclioptionsclusteradm.ConvertToFeatureGateAPI(genericclioptionsclusteradm.HubMutableFeatureGate, ocmfeature.DefaultHubWorkFeatureGates),
	}

	versionBundle, err := version.GetVersionBundle(o.bundleVersion)

	if err != nil {
		klog.Errorf("unable to retrieve version ", err)
		return err
	}

	o.values.BundleVersion = BundleVersion{
		RegistrationImageVersion: versionBundle.Registration,
		PlacementImageVersion:    versionBundle.Placement,
		WorkImageVersion:         versionBundle.Work,
		OperatorImageVersion:     versionBundle.Operator,
	}

	return nil
}

func (o *Options) validate() error {
	if o.force {
		return nil
	}
	// preflight check
	f := o.ClusteradmFlags.KubectlFactory
	kubeClient, _, _, err := helpers.GetClients(f)
	if err != nil {
		return err
	}
	if err := preflightinterface.RunChecks(
		[]preflightinterface.Checker{
			preflight.HubApiServerCheck{
				ClusterCtx: o.ClusteradmFlags.Context,
				ConfigPath: "", // TODO(@Promacanthus)： user custom kubeconfig path from command line arguments.
			},
			preflight.ClusterInfoCheck{
				Namespace:    metav1.NamespacePublic,
				ResourceName: preflight.BootstrapConfigMap,
				ClusterCtx:   o.ClusteradmFlags.Context,
				ConfigPath:   "", // TODO(@Promacanthus)： user custom kubeconfig path from command line arguments.
				Client:       kubeClient,
			},
		}, os.Stderr); err != nil {
		return err
	}

	if len(o.registry) == 0 {
		return fmt.Errorf("registry should not be empty")
	}

	// If --wait is set, some information during initialize process will print to output, the output would not keep
	// machine readable, so this behavior should be disabled
	if o.wait && o.output != "text" {
		return fmt.Errorf("output should be text if --wait is set")
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

	applierBuilder := apply.NewApplierBuilder()
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

	out, err = applier.ApplyDeployments(reader, o.values, o.ClusteradmFlags.DryRun, "", "init/operator.yaml")
	if err != nil {
		return err
	}
	output = append(output, out...)

	if !o.ClusteradmFlags.DryRun {
		if err := helperwait.WaitUntilCRDReady(apiExtensionsClient, "clustermanagers.operator.open-cluster-management.io", o.wait); err != nil {
			return err
		}
	}
	if o.wait && !o.ClusteradmFlags.DryRun {
		if err := helperwait.WaitUntilRegistrationOperatorReady(
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
		if err := helperwait.WaitUntilClusterManagerRegistrationReady(
			o.ClusteradmFlags.KubectlFactory,
			int64(o.ClusteradmFlags.Timeout)); err != nil {
			return err
		}
	}

	//if service-account wait for the sa secret
	if !o.useBootstrapToken && !o.ClusteradmFlags.DryRun {
		token, err = helpers.GetBootstrapTokenFromSA(context.TODO(), kubeClient)
		if err != nil {
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
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(sh, "%s --cluster-name $1", cmd)
		if err != nil {
			return err
		}
		if err := sh.Close(); err != nil {
			return err
		}
	}

	if o.output == "json" {
		err := clusteradmjson.WriteJsonOutput(os.Stdout, clusteradmjson.HubInfo{
			HubToken:     token,
			HubApiserver: restConfig.Host,
		})
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("The multicluster hub control plane has been initialized successfully!\n\n"+
			"You can now register cluster(s) to the hub control plane. Log onto those cluster(s) and run the following command:\n\n"+
			"    %s --cluster-name <cluster_name>\n\n"+
			"Replace <cluster_name> with a cluster name of your choice. For example, cluster1.\n\n",
			cmd,
		)
	}

	return apply.WriteOutput(o.outputFile, output)
}
