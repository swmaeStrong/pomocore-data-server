package structure

import "container/list"

type AhoCorasick struct {
	root *node
}

func NewAhoCorasick() *AhoCorasick {
	return &AhoCorasick{
		root: newNode(),
	}
}

func (a *AhoCorasick) Insert(pattern, category string) {
	now := a.root

	for _, r := range []rune(pattern) {
		if _, exists := now.children[r]; !exists {
			now.children[r] = newNode()
		}
		now = now.children[r]
	}
	now.categoryId = category
}

func (a *AhoCorasick) Connect() {
	q := list.New()
	q.PushBack(a.root)
	for q.Len() > 0 {
		now := q.Remove(q.Front()).(*node)
		for key, next := range now.children {
			if now == a.root {
				next.fail = a.root
			} else {
				dst := now.fail
				for dst != a.root && dst.children[key] == nil {
					dst = dst.fail
				}
				if child, exists := dst.children[key]; exists {
					dst = child
				}
				next.fail = dst
			}
			if next.categoryId == "" && next.fail.categoryId != "" {
				next.categoryId = next.fail.categoryId
			}
			q.PushBack(next)
		}
	}
}

func (a *AhoCorasick) Search(pattern string) string {
	now := a.root

	for _, r := range []rune(pattern) {
		for now != a.root && now.children[r] == nil {
			now = now.fail
		}

		if child, exists := now.children[r]; exists {
			now = child

			if now.categoryId != "" {
				return now.categoryId
			}

			temp := now.fail
			for temp != nil && temp != a.root && temp.categoryId == "" {
				temp = temp.fail
			}
			if temp != nil && temp.categoryId != "" {
				return temp.categoryId
			}
		}
	}
	return ""
}
