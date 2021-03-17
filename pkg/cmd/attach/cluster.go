// Copyright Contributors to the Open Cluster Management project
package attach

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-cluster-management/cm-cli/pkg/bindata"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var attachClusteExample = `
# Attach a cluster
%[1]s cm attach cluster --values values.yaml

# Attach a cluster with overwritting the cluster name
%[1]s cm attach cluster --values values.yaml --name mycluster
`

const (
	attachClusterScenarioDirectory = "scenarios/attach"
)

var valuesTemplatePath = filepath.Join(attachClusterScenarioDirectory, "values-template.yaml")

type AttachClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
	values                  map[string]interface{}
	clusterName             string
	clusterServer           string
	clusterToken            string
	clusterKubeConfig       string
	importFile              string
}

func newAttachClusterOptions(streams genericclioptions.IOStreams) *AttachClusterOptions {
	return &AttachClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdImportCluster provides a cobra command wrapping NewCmdImportCluster
func NewCmdAttachCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newAttachClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Import a cluster",
		Example:      fmt.Sprintf(attachClusteExample, os.Args[0]),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.SetUsageTemplate(applierscenarios.UsageTempate(cmd, valuesTemplatePath))
	cmd.Flags().StringVar(&o.clusterName, "name", "", "Name of the cluster to import")
	cmd.Flags().StringVar(&o.clusterServer, "cluster-server", "", "cluster server url of the cluster to import")
	cmd.Flags().StringVar(&o.clusterToken, "cluster-token", "", "token to access the cluster to import")
	cmd.Flags().StringVar(&o.clusterKubeConfig, "cluster-kubeconfigr", "", "path to the kubeconfig the cluster to import")
	cmd.Flags().StringVar(&o.importFile, "import-file", "", "the file which will contain the import secret for manual import")

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *AttachClusterOptions) Complete(cmd *cobra.Command, args []string) (err error) {
	o.values, err = apply.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	if o.clusterKubeConfig == "" {
		if ikubeConfig, ok := o.values["kubeConfig"]; ok {
			o.clusterKubeConfig = ikubeConfig.(string)
		}
	}
	o.values["kubeConfig"] = o.clusterKubeConfig

	if o.clusterServer == "" {
		if iclusterServer, ok := o.values["server"]; ok {
			o.clusterServer = iclusterServer.(string)
		}
	}
	o.values["server"] = o.clusterServer

	if o.clusterToken == "" {
		if iclusterToken, ok := o.values["token"]; ok {
			o.clusterToken = iclusterToken.(string)
		}
	}
	o.values["token"] = o.clusterToken

	return nil
}

func (o *AttachClusterOptions) Validate() error {
	if o.clusterName == "" {
		iname, ok := o.values["managedClusterName"]
		if !ok {
			return fmt.Errorf("cluster name is missing")
		}
		o.clusterName = iname.(string)
		if len(o.clusterName) == 0 {
			return fmt.Errorf("managedClusterName not specified")
		}
	}

	o.values["managedClusterName"] = o.clusterName

	if o.clusterName != "local-cluster" {
		if o.clusterKubeConfig != "" && (o.clusterToken != "" || o.clusterServer != "") {
			return fmt.Errorf("server/token and kubeConfig are mutually exclusif")
		}

		if (o.clusterToken == "" && o.clusterServer != "") ||
			(o.clusterToken != "" && o.clusterServer == "") {
			return fmt.Errorf("server or token is missing or should be removed")
		}

		if o.applierScenariosOptions.OutFile == "" &&
			o.clusterKubeConfig == "" &&
			o.clusterToken == "" &&
			o.clusterServer == "" &&
			o.importFile == "" {
			return fmt.Errorf("either kubeConfig or token/server or import-file must be provided")
		}
	}

	return nil
}

func (o *AttachClusterOptions) Run() (err error) {
	reader := bindata.NewBindataReader()

	applyOptions := &apply.ApplyOptions{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	err = applyOptions.ApplyWithValues(reader,
		filepath.Join(attachClusterScenarioDirectory, "hub"),
		o.values)
	if err != nil {
		return err
	}

	if o.importFile != "" {
		if o.applierScenariosOptions.OutFile == "" || o.clusterName == "local-cluster" {
			time.Sleep(10 * time.Second)
			client, err := helpers.GetClientFromFlags()
			if err != nil {
				return err
			}
			importSecret := &corev1.Secret{}
			err = client.Get(context.TODO(),
				types.NamespacedName{Name: fmt.Sprintf("%s-import", o.clusterName),
					Namespace: o.clusterName}, importSecret)
			if err != nil {
				return err
			}

			ys, err := yaml.Marshal(importSecret)
			if err != nil {
				return err
			}

			valueys := make(map[string]interface{})
			err = yaml.Unmarshal(ys, &valueys)
			if err != nil {
				return err
			}

			applyOptions.Silent = true
			applyOptions.OutFile = o.importFile
			err = applyOptions.ApplyWithValues(reader,
				filepath.Join(attachClusterScenarioDirectory, "managedcluster"),
				valueys)
			if err != nil {
				return err
			}
			if !o.applierScenariosOptions.Silent {
				fmt.Printf("Execute this command on the managed cluster\noc cm applier -d %s\n", o.importFile)
			}
		}
	}
	return nil
}
