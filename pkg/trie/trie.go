package trie

import (
	"path"
	"strings"
)

// PathTrie is a trie of paths with string keys and T values.

// PathTrie is a trie of string keys and T values. Internal nodes
// have nil values so stored nil values cannot be distinguished and are
// excluded from walks. By default, PathTrie will segment keys by forward
// slashes with PathSegmenter (e.g. "/a/b/c" -> "/a", "/b", "/c"). A custom
// StringSegmenter may be used to customize how strings are segmented into
// nodes. A classic trie might segment keys by rune (i.e. unicode points).
type PathTrie[T any] struct {
	segmenter StringSegmenter // key segmenter, must not cause heap allocs
	value     T
	children  map[string]*PathTrie[T]
	hasValue  bool
}

// PathTrieConfig for building a path trie with different segmenter
type PathTrieConfig struct {
	Segmenter StringSegmenter
}

// NewPathTrie allocates and returns a new *PathTrie.
func NewPathTrie[T any]() *PathTrie[T] {
	return &PathTrie[T]{
		segmenter: PathSegmenter,
	}
}

// NewPathTrieWithConfig allocates and returns a new *PathTrie with the given *PathTrieConfig
func NewPathTrieWithConfig[T any](config *PathTrieConfig) *PathTrie[T] {
	segmenter := PathSegmenter
	if config != nil && config.Segmenter != nil {
		segmenter = config.Segmenter
	}

	return &PathTrie[T]{
		segmenter: segmenter,
	}
}

// newPathTrieFromTrie returns new trie while preserving its config
func (trie *PathTrie[T]) newPathTrie() *PathTrie[T] {
	return &PathTrie[T]{
		segmenter: trie.segmenter,
	}
}

func (trie *PathTrie[T]) setValue(value T) {
	trie.value = value
	trie.hasValue = true
}
func (trie *PathTrie[T]) clearValue() {
	var d T
	trie.value = d
	trie.hasValue = false
}

// Get returns the value matched by path. Returns default value for internal
// nodes or for nodes with a default value.
func (trie *PathTrie[T]) Get(key string) (T, string) {
	node := trie
	var pre *PathTrie[T]
	var prefix []string
	for part, i := trie.segmenter(key, 0); part != ""; part, i = trie.segmenter(key, i) {
		node = node.children[part]
		if node == nil {
			break
		}
		prefix = append(prefix, part)
		pre = node
	}
	var d T
	if pre == nil {
		//not found
		return d, ""
	}
	return pre.value, strings.TrimPrefix(strings.TrimPrefix(key, path.Join(prefix...)), "/")
}

// Put inserts the value into the trie at the given key, replacing any
// existing items. It returns true if the put adds a new value, false
// if it replaces an existing value.
// Note that internal nodes have nil values so a stored nil value will not
// be distinguishable and will not be included in Walks.
func (trie *PathTrie[T]) Put(key string, value T) bool {
	node := trie
	for part, i := trie.segmenter(key, 0); part != ""; part, i = trie.segmenter(key, i) {
		child, _ := node.children[part]
		if child == nil {
			if node.children == nil {
				node.children = map[string]*PathTrie[T]{}
			}
			child = trie.newPathTrie()
			node.children[part] = child
		}
		node = child
	}
	// does node have an existing value?
	isNewVal := !node.hasValue
	node.setValue(value)
	return isNewVal
}

// Delete removes the value associated with the given key. Returns true if a
// node was found for the given key. If the node or any of its ancestors
// becomes childless as a result, it is removed from the trie.
func (trie *PathTrie[T]) Delete(key string) bool {
	var path []nodeStr[T] // record ancestors to check later
	node := trie
	for part, i := trie.segmenter(key, 0); part != ""; part, i = trie.segmenter(key, i) {
		path = append(path, nodeStr[T]{part: part, node: node})
		node = node.children[part]
		if node == nil {
			// node does not exist
			return false
		}
	}
	// delete the node value
	node.clearValue()
	// if leaf, remove it from its parent's children map. Repeat for ancestor path.
	if node.isLeaf() {
		// iterate backwards over path
		for i := len(path) - 1; i >= 0; i-- {
			parent := path[i].node
			part := path[i].part
			delete(parent.children, part)
			if !parent.isLeaf() {
				// parent has other children, stop
				break
			}
			parent.children = nil
			if parent.hasValue {
				// parent has a value, stop
				break
			}
		}
	}
	return true // node (internal or not) existed and its value was nil'd
}

// Walk iterates over each key/value stored in the trie and calls the given
// walker function with the key and value. If the walker function returns
// an error, the walk is aborted.
// The traversal is depth first with no guaranteed order.
func (trie *PathTrie[T]) Walk(walker WalkFunc[T]) error {
	return trie.walk("", walker)
}

// WalkPath iterates over each key/value in the path in trie from the root to
// the node at the given key, calling the given walker function for each
// key/value. If the walker function returns an error, the walk is aborted.
func (trie *PathTrie[T]) WalkPath(key string, walker WalkFunc[T]) error {
	// Get root value if one exists.
	if trie.hasValue {
		if err := walker("", trie.value); err != nil {
			return err
		}
	}
	for part, i := trie.segmenter(key, 0); ; part, i = trie.segmenter(key, i) {
		if trie = trie.children[part]; trie == nil {
			return nil
		}
		if trie.hasValue {
			var k string
			if i == -1 {
				k = key
			} else {
				k = key[0:i]
			}
			if err := walker(k, trie.value); err != nil {
				return err
			}
		}
		if i == -1 {
			break
		}
	}
	return nil
}

// PathTrie node and the part string key of the child the path descends into.
type nodeStr[T any] struct {
	node *PathTrie[T]
	part string
}

func (trie *PathTrie[T]) walk(key string, walker WalkFunc[T]) error {
	if trie.hasValue {
		if err := walker(key, trie.value); err != nil {
			return err
		}
	}
	for part, child := range trie.children {
		if err := child.walk(key+part, walker); err != nil {
			return err
		}
	}
	return nil
}

func (trie *PathTrie[T]) isLeaf() bool {
	return len(trie.children) == 0
}
