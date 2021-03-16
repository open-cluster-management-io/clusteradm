// Copyright Contributors to the Open Cluster Management project

package apply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/ghodss/yaml"
	"github.com/open-cluster-management/library-go/pkg/applier"
	"github.com/open-cluster-management/library-go/pkg/templateprocessor"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var ApplyExample = `
# Import a cluster
%[1]s --values values.yaml
`

type ApplyOptions struct {
	ConfigFlags *genericclioptions.ConfigFlags

	OutFile    string
	Directory  string
	ValuesPath string
	DryRun     bool
	Prefix     string
	Delete     bool
	Timeout    int
	Force      bool
	Silent     bool

	genericclioptions.IOStreams
}

func NewApplierOptions(streams genericclioptions.IOStreams) *ApplyOptions {
	return &ApplyOptions{
		ConfigFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

//AddFlags returns a flagset
func (o *ApplyOptions) AddFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&o.OutFile, "output", "o", "",
		"Output file. If set nothing will be applied but a file will be generate "+
			"which you can apply later with 'kubectl <create|apply|delete> -f")
	flagSet.StringVarP(&o.Directory, "directory", "d", "", "The directory or file containing the template(s)")
	flagSet.StringVar(&o.ValuesPath, "values", "", "The file containing the values")
	flagSet.BoolVar(&o.DryRun, "dry-run", false, "if set only the rendered yaml will be shown, default false")
	flagSet.StringVarP(&o.Prefix, "prefix", "p", "", "The prefix to add to each value names, for example 'Values'")
	flagSet.BoolVar(&o.Delete, "delete", false,
		"if set only the resource defined in the yamls will be deleted, default false")
	flagSet.IntVar(&o.Timeout, "timout", 5, "Timeout in second to apply one resource, default 5 sec")
	flagSet.BoolVarP(&o.Force, "force", "f", false, "If set, the finalizers will be removed before delete")
	flagSet.BoolVar(&o.Silent, "silent", false, "If set the applier will run silently")
}

//Complete retrieve missing options
func (o *ApplyOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

//Validate validates the options
func (o *ApplyOptions) Validate() error {
	return o.CheckOptions()
}

//Run runs the commands
func (o *ApplyOptions) Run() error {
	return o.Apply()
}

//Apply applies the resources
func (o *ApplyOptions) Apply() (err error) {

	values, err := ConvertValuesFileToValuesMap(o.ValuesPath, o.Prefix)
	if err != nil {
		return err
	}

	templateReader := templateprocessor.NewYamlFileReader(o.Directory)

	return o.ApplyWithValues(templateReader, "", values)
}

func ConvertValuesFileToValuesMap(path, prefix string) (values map[string]interface{}, err error) {
	var b []byte
	if path != "" {
		b, err = ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, err
		}
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeNamedPipe != 0 {
		b = append(b, '\n')
		pdata, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		b = append(b, pdata...)
	}

	valuesc := make(map[string]interface{})
	err = yaml.Unmarshal(b, &valuesc)
	if err != nil {
		return nil, err
	}

	values = make(map[string]interface{})
	if prefix != "" {
		values[prefix] = valuesc
	} else {
		values = valuesc
	}

	klog.V(4).Infof("values:\n%v", values)

	return values, nil
}

func (o *ApplyOptions) ApplyWithValues(templateReader templateprocessor.TemplateReader, path string, values map[string]interface{}) (err error) {
	if o.OutFile != "" {
		templateProcessor, err := templateprocessor.NewTemplateProcessor(templateReader, &templateprocessor.Options{})
		if err != nil {
			return err
		}
		outV, err := templateProcessor.TemplateResourcesInPathYaml(path, []string{}, true, values)
		if err != nil {
			return err
		}
		out := templateprocessor.ConvertArrayOfBytesToString(outV)
		klog.V(1).Infof("result:\n%s", out)
		return ioutil.WriteFile(filepath.Clean(o.OutFile), []byte(templateprocessor.ConvertArrayOfBytesToString(outV)), 0600)
	}

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
	if err != nil {
		return err
	}

	applierOptions := &applier.Options{
		Backoff: &wait.Backoff{
			Steps:    4,
			Duration: 500 * time.Millisecond,
			Factor:   5.0,
			Jitter:   0.1,
			Cap:      time.Duration(o.Timeout) * time.Second,
		},
		DryRun:      o.DryRun,
		ForceDelete: o.Force,
	}
	if o.DryRun {
		client = crclient.NewDryRunClient(client)
	}
	a, err := applier.NewApplier(templateReader,
		&templateprocessor.Options{},
		client,
		nil,
		nil,
		applier.DefaultKubernetesMerger,
		applierOptions)
	if err != nil {
		return err
	}
	if o.Delete {
		err = a.DeleteInPath(path, nil, true, values)
	} else {
		err = a.CreateOrUpdateInPath(path, nil, true, values)
	}
	if err != nil {
		return err
	}
	return nil
}

//CheckOptions checks the options
func (o *ApplyOptions) CheckOptions() error {
	klog.V(2).Infof("-d: %s", o.Directory)
	if o.Directory == "" {
		return fmt.Errorf("-d must be set")
	}
	if o.OutFile != "" &&
		(o.DryRun || o.Delete || o.Force) {
		return fmt.Errorf("-o is not compatible with -dry-run, delete or force")
	}
	return nil
}
