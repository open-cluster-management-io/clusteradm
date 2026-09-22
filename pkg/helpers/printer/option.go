// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"fmt"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
)

type PrinterOption struct {
	Options printers.PrintOptions
	Format  string
	tree    TreePrinter
	table   printers.ResourcePrinter
	yaml    printers.YAMLPrinter

	treeConverter  func(runtime.Object, *TreePrinter) (*TreePrinter, error)
	tableConverter func(runtime.Object) (*metav1.Table, error)
}

func NewPrinterOption(o printers.PrintOptions) *PrinterOption {
	return &PrinterOption{
		Options: o,
	}
}

func (p *PrinterOption) AddFlag(fs *pflag.FlagSet) {
	fs.StringVarP(&p.Format, "output", "o", "tree", "output format can be tree, table or yaml")
}

func (p *PrinterOption) Complete() {
	p.tree = NewTreePrinter(p.Options.Kind.Kind)
	p.table = printers.NewTablePrinter(p.Options)
	p.yaml = printers.YAMLPrinter{}
}

func (p *PrinterOption) Validate() error {
	if p.Format != "tree" && p.Format != "table" && p.Format != "yaml" {
		return fmt.Errorf("invalid output format")
	}
	return nil
}

func (p *PrinterOption) WithTreeConverter(f func(runtime.Object, *TreePrinter) (*TreePrinter, error)) *PrinterOption {
	p.treeConverter = f
	return p
}
func (p *PrinterOption) WithTableConverter(f func(runtime.Object) (*metav1.Table, error)) *PrinterOption {
	p.tableConverter = f
	return p
}

func (p *PrinterOption) Print(stream genericiooptions.IOStreams, obj runtime.Object) error {
	switch p.Format {
	case "tree":
		tree, err := p.treeConverter(obj, &p.tree)
		if err != nil {
			return err
		}
		return tree.Print(stream.Out)
	case "table":
		table, err := p.tableConverter(obj)
		if err != nil {
			return err
		}
		return p.table.PrintObj(table, stream.Out)
	case "yaml":
		objs, err := meta.ExtractList(obj)
		if err != nil {
			return err
		}

		for _, item := range objs {
			err := p.yaml.PrintObj(item, stream.Out)
			if err != nil {
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("invalid output format")
	}
}
