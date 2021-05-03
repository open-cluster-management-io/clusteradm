// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	appliercmd "github.com/open-cluster-management/applier/pkg/applier/cmd"
	"github.com/open-cluster-management/cm-cli/pkg/cmd/attach/cluster/scenario"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"

	"github.com/spf13/cobra"
)

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}

	//Check if default values must be used
	if o.applierScenariosOptions.ValuesPath == "" {
		if o.clusterName != "" {
			reader := scenario.GetApplierScenarioResourcesReader()
			b, err := reader.Asset(valuesDefaultPath)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(b, &o.values)
			if err != nil {
				return err
			}
			mc := o.values["managedCluster"].(map[string]interface{})
			mc["name"] = o.clusterName
		} else {
			return fmt.Errorf("values or name are missing")
		}
	} else {
		//Read values
		o.values, err = appliercmd.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
		if err != nil {
			return err
		}
	}

	imc, ok := o.values["managedCluster"]
	if !ok || imc == nil {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})

	if o.clusterKubeConfig == "" {
		if ikubeConfig, ok := mc["kubeConfig"]; ok {
			o.clusterKubeConfig = ikubeConfig.(string)
		}
	} else {
		b, err := ioutil.ReadFile(o.clusterKubeConfig)
		if err != nil {
			return err
		}
		o.clusterKubeConfig = string(b)
	}

	mc["kubeConfig"] = o.clusterKubeConfig

	if o.clusterServer == "" {
		if iclusterServer, ok := mc["server"]; ok {
			o.clusterServer = iclusterServer.(string)
		}
	}
	mc["server"] = o.clusterServer

	if o.clusterToken == "" {
		if iclusterToken, ok := mc["token"]; ok {
			o.clusterToken = iclusterToken.(string)
		}
	}
	mc["token"] = o.clusterToken

	return nil
}

func (o *Options) validate() error {
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.validateWithClient(client)
}

func (o *Options) validateWithClient(client crclient.Client) error {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		return nil
	}

	imc, ok := o.values["managedCluster"]
	if !ok || imc == nil {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})

	if o.clusterName == "" {
		iname, ok := mc["name"]
		if !ok || iname == nil {
			return fmt.Errorf("cluster name is missing")
		}
		o.clusterName = iname.(string)
		if len(o.clusterName) == 0 {
			return fmt.Errorf("managedCluster.name not specified")
		}
	}

	mc["name"] = o.clusterName

	if o.clusterName != "local-cluster" {
		if o.clusterKubeConfig != "" && (o.clusterToken != "" || o.clusterServer != "") {
			return fmt.Errorf("server/token and kubeConfig are mutually exclusif")
		}

		if (o.clusterToken == "" && o.clusterServer != "") ||
			(o.clusterToken != "" && o.clusterServer == "") {
			return fmt.Errorf("server or token is missing or should be removed")
		}

		cd := unstructured.Unstructured{}
		cd.SetKind("ClusterDeployment")
		cd.SetAPIVersion("hive.openshift.io/v1")
		err := client.Get(context.TODO(),
			crclient.ObjectKey{
				Name:      o.clusterName,
				Namespace: o.clusterName,
			}, &cd)

		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		} else {
			o.hiveScenario = true
		}

		if o.applierScenariosOptions.OutFile == "" &&
			o.clusterKubeConfig == "" &&
			o.clusterToken == "" &&
			o.clusterServer == "" &&
			o.importFile == "" &&
			!o.hiveScenario {
			return fmt.Errorf("either kubeConfig or token/server or import-file must be provided")
		}
	}

	return nil
}

func (o *Options) run() (err error) {
	if o.applierScenariosOptions.OutTemplatesDir != "" {
		reader := scenario.GetApplierScenarioResourcesReader()
		return reader.ExtractAssets(scenarioDirectory, o.applierScenariosOptions.OutTemplatesDir)
	}
	client, err := helpers.GetControllerRuntimeClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) (err error) {
	reader := scenario.GetApplierScenarioResourcesReader()

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

	if !o.hiveScenario &&
		o.importFile != "" &&
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
			return nil
		}
	}
	return nil
}
