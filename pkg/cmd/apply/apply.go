// Copyright Contributors to the Open Cluster Management project

package apply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/ghodss/yaml"
	"github.com/open-cluster-management/cm-cli/pkg/helpers"
	"github.com/open-cluster-management/library-go/pkg/applier"
	"github.com/open-cluster-management/library-go/pkg/templateprocessor"

	// "k8s.io/klog"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var example = `
# Import a cluster
%[1]s --values values.yaml
`

type Options struct {
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

func newOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		ConfigFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

func NewCmd(streams genericclioptions.IOStreams) *cobra.Command {
	o := newOptions(streams)

	cmd := &cobra.Command{
		Use:          "applier",
		Short:        "apply templates",
		Example:      fmt.Sprintf(example, os.Args[0]),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}
			if err := o.run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&o.OutFile, "output", "o", "",
		"Output file. If set nothing will be applied but a file will be generate "+
			"which you can apply later with 'kubectl <create|apply|delete> -f")
	cmd.Flags().StringVarP(&o.Directory, "directory", "d", "", "The directory or file containing the template(s)")
	cmd.Flags().StringVar(&o.ValuesPath, "values", "", "The file containing the values")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "if set only the rendered yaml will be shown, default false")
	cmd.Flags().StringVarP(&o.Prefix, "prefix", "p", "", "The prefix to add to each value names, for example 'Values'")
	cmd.Flags().BoolVar(&o.Delete, "delete", false,
		"if set only the resource defined in the yamls will be deleted, default false")
	cmd.Flags().IntVar(&o.Timeout, "timout", 5, "Timeout in second to apply one resource, default 5 sec")
	cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "If set, the finalizers will be removed before delete")
	cmd.Flags().BoolVar(&o.Silent, "silent", false, "If set the applier will run silently")

	o.ConfigFlags.AddFlags(cmd.Flags())

	return cmd
}

//complete retrieve missing options
func (o *Options) complete(cmd *cobra.Command, args []string) error {
	return nil
}

//validate validates the options
func (o *Options) validate() error {
	return o.checkOptions()
}

//run runs the commands
func (o *Options) run() error {
	client, err := helpers.GetClientFromFlags(o.ConfigFlags)
	if err != nil {
		return err
	}

	return o.apply(client)
}

// func (o *Options) discardKlogOutput() {
// 	// if o.OutFile != "" {
// 	klog.SetOutput(ioutil.Discard)
// 	// }
// }

//apply applies the resources
func (o *Options) apply(client crclient.Client) (err error) {

	// o.discardKlogOutput()

	values, err := ConvertValuesFileToValuesMap(o.ValuesPath, o.Prefix)
	if err != nil {
		return err
	}

	templateReader := templateprocessor.NewYamlFileReader(o.Directory)

	return o.ApplyWithValues(client, templateReader, "", values)
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

	// klog.V(4).Infof("values:\n%v", values)

	return values, nil
}

func (o *Options) ApplyWithValues(client crclient.Client, templateReader templateprocessor.TemplateReader, path string, values map[string]interface{}) (err error) {
	if o.OutFile != "" {
		templateProcessor, err := templateprocessor.NewTemplateProcessor(templateReader, &templateprocessor.Options{})
		if err != nil {
			return err
		}
		outV, err := templateProcessor.TemplateResourcesInPathYaml(path, []string{}, true, values)
		if err != nil {
			return err
		}
		// out := templateprocessor.ConvertArrayOfBytesToString(outV)
		// klog.V(1).Infof("result:\n%s", out)
		return ioutil.WriteFile(filepath.Clean(o.OutFile), []byte(templateprocessor.ConvertArrayOfBytesToString(outV)), 0600)
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

//checkOptions checks the options
func (o *Options) checkOptions() error {
	// klog.V(2).Infof("-d: %s", o.Directory)
	if o.Directory == "" {
		return fmt.Errorf("-d must be set")
	}
	if o.OutFile != "" &&
		(o.DryRun || o.Delete || o.Force) {
		return fmt.Errorf("-o is not compatible with -dry-run, delete or force")
	}
	return nil
}
