// Copyright Contributors to the Open Cluster Management project
package printer

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

func TestDefaultSegmenter(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		start       int
		wantSegment string
		wantNext    int
	}{
		{name: "empty path", path: "", start: 0, wantSegment: "", wantNext: -1},
		{name: "start negative", path: "a.b", start: -1, wantSegment: "", wantNext: -1},
		{name: "start past end", path: "a.b", start: 3, wantSegment: "", wantNext: -1},
		{name: "single segment no dot", path: "a", start: 0, wantSegment: "a", wantNext: -1},
		{name: "first of two segments", path: "a.b", start: 0, wantSegment: "a", wantNext: 1},
		{name: "second of two segments keeps leading dot", path: "a.b", start: 1, wantSegment: ".b", wantNext: -1},
		{name: "three segments first", path: "a.b.c", start: 0, wantSegment: "a", wantNext: 1},
		{name: "three segments middle", path: "a.b.c", start: 1, wantSegment: ".b", wantNext: 3},
		{name: "three segments last", path: "a.b.c", start: 3, wantSegment: ".c", wantNext: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			segment, next := DefaultSegmenter(c.path, c.start)
			if segment != c.wantSegment {
				t.Errorf("segment: got %q, want %q", segment, c.wantSegment)
			}
			if next != c.wantNext {
				t.Errorf("next: got %d, want %d", next, c.wantNext)
			}
		})
	}
}

func TestTriePutGet(t *testing.T) {
	trie := NewTrie(DefaultSegmenter)

	isNew := trie.Put("a.b", "first")
	if !isNew {
		t.Fatalf("expected first put of a.b to report isNew=true")
	}

	if got := trie.Get("a.b"); got != "first" {
		t.Fatalf("got %v, want %q", got, "first")
	}

	isNew = trie.Put("a.b", "second")
	if isNew {
		t.Fatalf("expected overwrite of a.b to report isNew=false")
	}
	if got := trie.Get("a.b"); got != "second" {
		t.Fatalf("got %v, want %q", got, "second")
	}

	if got := trie.Get("a.c"); got != nil {
		t.Fatalf("expected missing key to return nil, got %v", got)
	}
}

func TestTrieGetOnEmptyTrie(t *testing.T) {
	trie := NewTrie(DefaultSegmenter)
	if got := trie.Get("anything"); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestTrieIsLeaf(t *testing.T) {
	trie := NewTrie(DefaultSegmenter)
	if !trie.isLeaf() {
		t.Fatalf("expected a freshly created trie to be a leaf")
	}

	trie.Put("a.b", "value")
	if trie.isLeaf() {
		t.Fatalf("expected trie with a child to not be a leaf")
	}
}

func TestTrieIter(t *testing.T) {
	trie := NewTrie(DefaultSegmenter)
	trie.Put("a.b", "first")
	trie.Put("a.c", "second")
	trie.Put("d", "third")

	var got []string
	err := trie.Iter(func(key string, value interface{}) error {
		got = append(got, key+"="+value.(string))
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a.b=first", "a.c=second", "d=third"}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTrieIterPropagatesError(t *testing.T) {
	trie := NewTrie(DefaultSegmenter)
	trie.Put("a.b", "value")

	wantErr := errors.New("stop")
	err := trie.Iter(func(key string, value interface{}) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected iter to propagate the callback error, got %v", err)
	}
}
