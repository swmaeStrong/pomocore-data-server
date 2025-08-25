package structure

type Trie struct {
	root *node
}

func NewTrie() *Trie {
	return &Trie{
		root: newNode(),
	}
}

func (t *Trie) Insert(word, category string) {
	t.insert(word, category)
}

func (t *Trie) insert(word, category string) {
	now := t.root
	for _, r := range []rune(word) {
		if _, exists := now.children[r]; !exists {
			now.children[r] = newNode()
		}
		now = now.children[r]
	}
	now.categoryId = category
}

func (t *Trie) Search(word string) string {
	now := t.root
	for _, r := range []rune(word) {
		if _, exists := now.children[r]; !exists {
			return ""
		}
		now = now.children[r]
	}
	return now.categoryId
}
