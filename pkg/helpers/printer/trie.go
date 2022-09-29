// Copyright Contributors to the Open Cluster Management project
package printer

import "strings"

type IterFunc func(key string, value interface{}) error

func DefaultIterFunc(key string, value interface{}) error {
	return nil
}

type Segmenter func(key string, start int) (segment string, nextIndex int)

func DefaultSegmenter(path string, start int) (segment string, next int) {
	if len(path) == 0 || start < 0 || start > len(path)-1 {
		return "", -1
	}
	end := strings.IndexRune(path[start+1:], '.')
	if end == -1 {
		return path[start:], -1
	}
	return path[start : start+end+1], start + end + 1
}

type Trier interface {
	Get(key string) interface{}
	Put(key string, value interface{}) bool
	Iter(it IterFunc) error
}

type Trie struct {
	segmenter Segmenter // key segmenter, must not cause heap allocs
	value     interface{}
	children  map[string]*Trie
}

func NewTrie(sgmt Segmenter) *Trie {
	return &Trie{
		segmenter: sgmt,
	}
}

func (trie *Trie) Get(key string) interface{} {
	node := trie
	for part, i := trie.segmenter(key, 0); part != ""; part, i = trie.segmenter(key, i) {
		node = node.children[part]
		if node == nil {
			return nil
		}
	}
	return node.value
}

func (trie *Trie) newTrie() *Trie {
	return &Trie{
		segmenter: trie.segmenter,
	}
}
func (trie *Trie) Put(key string, value interface{}) bool {
	node := trie
	for part, i := trie.segmenter(key, 0); part != ""; part, i = trie.segmenter(key, i) {
		child := node.children[part]
		if child == nil {
			if node.children == nil {
				node.children = map[string]*Trie{}
			}
			child = trie.newTrie()
			node.children[part] = child
		}
		node = child
	}

	isNewVal := node.value == nil
	node.value = value
	return isNewVal
}

func (trie *Trie) Iter(it IterFunc) error {
	return trie.iter("", it)
}

func (trie *Trie) iter(key string, it IterFunc) error {
	if trie.value != nil {
		if err := it(key, trie.value); err != nil {
			return err
		}
	}
	for part, child := range trie.children {
		if err := child.iter(key+part, it); err != nil {
			return err
		}
	}
	return nil
}

func (trie *Trie) isLeaf() bool {
	return len(trie.children) == 0
}
