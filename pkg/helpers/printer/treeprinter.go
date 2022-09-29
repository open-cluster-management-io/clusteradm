// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/disiqueira/gotree"
)

type TreePrinter struct {
	name string
	tree *Trie
}

func NewTreePrinter(name string) TreePrinter {
	return TreePrinter{
		name: name,
		tree: NewTrie(DefaultSegmenter),
	}
}

// AddFileds add the fileds need to print to TreePrinter.
// @param name : the name of object to be print
// @param mp : key is a string represents level and path, value is the value will be printed.
func (t *TreePrinter) AddFileds(name string, mp *map[string]interface{}) {
	if mp == nil {
		return
	}
	t.add(name, "")
	for key, value := range *mp {
		t.add(name+key, value)
	}
}

// add the key value to TreePrinter.
func (t *TreePrinter) add(key string, value interface{}) {
	t.tree.Put(key, value)
}

func (t *TreePrinter) Print(outstream io.Writer) error {
	_, err := fmt.Fprint(outstream, t.build().Print())
	return err
}

func (t *TreePrinter) build() gotree.Tree {
	var dfs func(part string, root *Trie) gotree.Tree

	dfs = func(part string, root *Trie) gotree.Tree {
		if root.value == nil {
			root.value = ""
		}
		line := fmt.Sprintf("<%s> %v", part, root.value)
		level0 := gotree.New(line)

		if root.isLeaf() {
			return level0
		}

		for key, value := range root.children {
			level0.AddTree(dfs(strings.TrimPrefix(key, "."), value))
		}
		return level0
	}

	return dfs(t.name, t.tree)
}
