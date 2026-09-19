// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"bytes"
	"strings"
	"testing"
)

func TestTreePrinterAddFieldsNilMap(t *testing.T) {
	tp := NewTreePrinter("root")
	tp.AddFileds("root", nil)

	buf := &bytes.Buffer{}
	if err := tp.Print(buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected some output even with a nil field map")
	}
}

func TestTreePrinterAddFieldsAndPrint(t *testing.T) {
	tp := NewTreePrinter("Cluster")
	fields := map[string]interface{}{
		".status":        "joined",
		".workload.pod1": "applied",
	}
	tp.AddFileds("cluster1", &fields)

	buf := &bytes.Buffer{}
	if err := tp.Print(buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"<Cluster>", "<cluster1>", "joined", "applied"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestTreePrinterMultipleObjects(t *testing.T) {
	tp := NewTreePrinter("Cluster")

	fields1 := map[string]interface{}{".status": "joined"}
	tp.AddFileds("cluster1", &fields1)

	fields2 := map[string]interface{}{".status": "pending"}
	tp.AddFileds("cluster2", &fields2)

	buf := &bytes.Buffer{}
	if err := tp.Print(buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	for _, want := range []string{"cluster1", "cluster2", "joined", "pending"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}
