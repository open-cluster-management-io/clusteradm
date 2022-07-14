// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"fmt"
	"io"

	"github.com/disiqueira/gotree"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

type PrinterInterface interface{}

type Printer struct {
	format string
}

func NewPrinter(f string) *Printer {
	return &Printer{
		format: f,
	}
}

func (p *Printer) IsTable() bool {
	return p.format == "table"
}

func (p *Printer) IsYaml() bool {
	return p.format == "yaml"
}

func (p *Printer) IsTree() bool {
	return p.format == "tree"
}

func (p *Printer) PrintObject(outstream io.Writer, obj PrinterInterface, tableprintoption printers.PrintOptions) error {
	switch p.format {
	case "table":
		return p.printTable(outstream, obj, tableprintoption)
	case "yaml":
		return p.printYaml(outstream, obj)
	case "tree":
		return p.printTree(outstream, obj)
	default:
		return fmt.Errorf("invalid format value %v, format must be one of table, yaml or tree", p.format)
	}
}

func (p *Printer) printTable(outstream io.Writer, obj PrinterInterface, printoption printers.PrintOptions) error {
	object, ok := obj.(runtime.Object)
	if !ok {
		return fmt.Errorf("obj pass to printTable() must be runtime.Object type")
	}

	return printers.NewTablePrinter(printoption).PrintObj(object, outstream)
}

func (p *Printer) printYaml(outstream io.Writer, obj PrinterInterface) error {
	object, ok := obj.(runtime.Object)
	if !ok {
		return fmt.Errorf("obj pass to printYaml() must be runtime.Object type")
	}

	yamlPrinter := &printers.YAMLPrinter{}
	return yamlPrinter.PrintObj(object, outstream)
}

func (p *Printer) printTree(outstream io.Writer, obj PrinterInterface) error {
	object, ok := obj.(gotree.Tree)
	if !ok {
		return fmt.Errorf("obj pass to printTree() must be gotree.Tree type")
	}

	fmt.Fprint(outstream, object.Print())
	return nil

}
