// Copyright Contributors to the Open Cluster Management project
package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"
	"github.com/open-cluster-management/cm-cli/pkg/resources"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
)

var testDir = filepath.Join("..", "..", "..", "..", "test", "unit")
var detachClusterTestDir = filepath.Join(testDir, "resources", "detach", "cluster")

func (o *Options) complete(cmd *cobra.Command, args []string) (err error) {
	o.values, err = apply.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	if len(o.values) == 0 {
		return fmt.Errorf("values are missing")
	}

	return nil
}

func (o *Options) validate() error {
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

	return nil
}

func (o *Options) run() error {
	client, err := helpers.GetClientFromFlags(o.applierScenariosOptions.ConfigFlags)
	if err != nil {
		return err
	}
	return o.runWithClient(client)
}

func (o *Options) runWithClient(client crclient.Client) error {
	reader := resources.NewResourcesReader()

	applyOptions := &apply.Options{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    true,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	return applyOptions.ApplyWithValues(client, reader,
		filepath.Join(detachClusterTestDir, "managed_cluster_cr.yaml"),
		o.values)

}
