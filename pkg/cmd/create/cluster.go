package create

import (
	"context"
	"fmt"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/apply"

	"github.com/ghodss/yaml"
	"github.com/open-cluster-management/cm-cli/pkg/bindata"
	"github.com/open-cluster-management/library-go/pkg/templateprocessor"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-cluster-management/cm-cli/pkg/cmd/applierscenarios"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	AWS     = "aws"
	AZURE   = "azure"
	GCP     = "gcp"
	VSPHERE = "vsphere"
)

const (
	createClusterScenarioDirectory = "scnearios/createdestroy/hub/common"
)

var createClusteExample = `
# Create a cluster
%[1]s cm create cluster --values values.yaml
`

type CreateClusterOptions struct {
	applierScenariosOptions *applierscenarios.ApplierScenariosOptions
	cloud                   string
	values                  map[string]interface{}
}

func newCreateClusterOptions(streams genericclioptions.IOStreams) *CreateClusterOptions {
	return &CreateClusterOptions{
		applierScenariosOptions: applierscenarios.NewApplierScenariosOptions(streams),
	}
}

// NewCmdCreateCluster ...
func NewCmdCreateCluster(streams genericclioptions.IOStreams) *cobra.Command {
	o := newCreateClusterOptions(streams)

	cmd := &cobra.Command{
		Use:          "cluster",
		Short:        "Create a cluster",
		Example:      fmt.Sprintf(createClusteExample, "oc/kubectl"),
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

	o.applierScenariosOptions.AddFlags(cmd.Flags())
	o.applierScenariosOptions.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *CreateClusterOptions) Complete(cmd *cobra.Command, args []string) (err error) {
	o.values, err = apply.ConvertValuesFileToValuesMap(o.applierScenariosOptions.ValuesPath, "")
	if err != nil {
		return err
	}

	return nil
}

func (o *CreateClusterOptions) Validate() (err error) {
	imc, ok := o.values["managedCluster"]
	if !ok {
		return fmt.Errorf("managedCluster is missing")
	}
	mc := imc.(map[string]interface{})
	icloud, ok := mc["cloud"]
	if !ok {
		return fmt.Errorf("cloud type is missing")
	}
	cloud := icloud.(string)
	if cloud != AWS && cloud != AZURE && cloud != GCP && cloud != VSPHERE {
		return fmt.Errorf("Supported cloud type are (%s, %s, %s, %s) and got %s", AWS, AZURE, GCP, VSPHERE, cloud)
	}
	o.cloud = cloud

	iname, ok := mc["name"]
	if !ok {
		return fmt.Errorf("cluster name is missing")
	}
	name := iname.(string)
	if len(name) == 0 {
		return fmt.Errorf("managedCluster.name not specified")
	}

	return nil
}

func (o *CreateClusterOptions) Run() error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}
	client, err := crclient.New(config, crclient.Options{})

	pullSecret := &corev1.Secret{}
	err = client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      "pull-secret",
			Namespace: "openshift-config",
		},
		pullSecret)
	if err != nil {
		return err
	}

	ps, err := yaml.Marshal(pullSecret)
	if err != nil {
		return err
	}

	valueps := make(map[string]interface{})
	err = yaml.Unmarshal(ps, &valueps)
	if err != nil {
		return err
	}

	o.values["pullSecret"] = valueps

	reader := bindata.NewBindataReader()
	tp, err := templateprocessor.NewTemplateProcessor(
		reader,
		&templateprocessor.Options{},
	)
	if err != nil {
		return err
	}

	installConfig, err := tp.TemplateResource("scenarios/createdestroy/hub/"+o.cloud+"/install_config.yaml", o.values)
	if err != nil {
		return err
	}

	valueic := make(map[string]interface{})
	err = yaml.Unmarshal(installConfig, &valueic)
	if err != nil {
		return err
	}

	o.values["installConfig"] = valueic

	applyOptions := &apply.ApplyOptions{
		OutFile:     o.applierScenariosOptions.OutFile,
		ConfigFlags: o.applierScenariosOptions.ConfigFlags,

		Delete:    false,
		Timeout:   o.applierScenariosOptions.Timeout,
		Force:     o.applierScenariosOptions.Force,
		Silent:    o.applierScenariosOptions.Silent,
		IOStreams: o.applierScenariosOptions.IOStreams,
	}

	return applyOptions.ApplyWithValues(reader, "scenarios/createdestroy/hub/common", o.values)
}
