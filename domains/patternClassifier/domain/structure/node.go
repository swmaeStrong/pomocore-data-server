package structure

type node struct {
	children   map[rune]*node
	fail       *node
	categoryId string
}

func newNode() *node {
	return &node{
		children: make(map[rune]*node),
	}
}
