// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"
	"github.com/open-cluster-management/cm-cli/pkg/resources"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}
	o.values, err = appliercmd.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	if len(o.values) == 0 {
		return fmt.Errorf("values are missing")
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

func (o *Options) validate() error {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}

	if o.clusterName == "" {
		iname, ok := o.values["managedClusterName"]
		if !ok || iname == nil {
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

func (o *Options) run() (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		reader := resources.NewResourcesReader()
		return reader.ExtractAssets(scenarioDirectory, o.applierScenariosOptions.OutTemplatesDir)
	}
	client, err := helpers.GetClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) (err error) {
	reader := resources.NewResourcesReader()

	applyOptions := &appliercmd.Options{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	err = applyOptions.ApplyWithValues(client, reader,
		filepath.Join(scenarioDirectory, "hub"),
		o.values)
	if err != nil {
		return err
	}

	if o.importFile != "" &&
		o.applierScenariosOptions.OutFile == "" &&
		o.clusterName != "local-cluster" {
		time.Sleep(10 * time.Second)
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
		err = applyOptions.ApplyWithValues(client, reader,
			filepath.Join(scenarioDirectory, "managedcluster"),
			valueys)
		if err != nil {
			return err
		}
		if !o.applierScenariosOptions.Silent {
			fmt.Printf("Execute this command on the managed cluster\n%s applier -d %s\n", helpers.GetExampleHeader(), o.importFile)
		}
	}
	return nil
}
