package main

/// Returns -1 of a < b, 0 if a == b and 1 if a > b
type Comparator[T any] func(a, b T) int

type Node[T any] struct {
	Data    T
	Balance int
	Link    [2]*Node[T]
}

type AVLTree[T any] struct {
	Root *Node[T]
	comp Comparator[T]
}

func NewAVLTree[T any](comp Comparator[T]) *AVLTree[T] {
	return &AVLTree[T]{ nil, comp }
}

func (tree *AVLTree[T]) singleRotate(root *Node[T], dir int) *Node[T] {
	save := root.Link[dir^1]
	root.Link[dir^1] = save.Link[dir]
	save.Link[dir] = root
	return save
}

func (tree *AVLTree[T]) doubleRotate(root *Node[T], dir int) *Node[T] {
	root.Link[dir^1] = tree.singleRotate(root.Link[dir^1], dir^1)
	return tree.singleRotate(root, dir)
}

func (tree *AVLTree[T]) adjustBalance(root *Node[T], dir, bal int) {
	n := root.Link[dir]
	nn := n.Link[dir^1]
	switch nn.Balance {
	case 0:
		root.Balance = 0
		n.Balance = 0
	case bal:
		root.Balance = -bal
		n.Balance = 0
	default:
		root.Balance = 0
		n.Balance = bal
	}
	nn.Balance = 0
}

func (tree *AVLTree[T]) insertBalance(root *Node[T], dir int) *Node[T] {
	n := root.Link[dir]
	bal := 2*dir - 1
	if n.Balance == bal {
		root.Balance = 0
		n.Balance = 0
		return tree.singleRotate(root, dir^1)
	}
	tree.adjustBalance(root, dir, bal)
	return tree.doubleRotate(root, dir^1)
}

func (tree *AVLTree[T]) insertRecursive(root *Node[T], data T) (*Node[T], bool) {
	if root == nil {
		return &Node[T]{Data: data}, false
	}
	c := tree.comp(root.Data, data)
	if c == 0 {
		return root, true
	}
	dir := 0
	if c < 0 {
		dir = 1
	}
	var done bool
	root.Link[dir], done = tree.insertRecursive(root.Link[dir], data)
	if done {
		return root, true
	}
	root.Balance += 2*dir - 1
	switch root.Balance {
	case 0:
		return root, true
	case 1, -1:
		return root, false
	}
	return tree.insertBalance(root, dir), true
}

func (tree *AVLTree[T]) Insert(data T) {
	tree.Root, _ = tree.insertRecursive(tree.Root, data)
}

func (tree *AVLTree[T]) removeRecursive(root *Node[T], data T) *Node[T] {
	if root == nil {
		return nil
	}

	dir := 0
	if tree.comp(data, root.Data) > 0 {
		dir = 1
	} else if tree.comp(data, root.Data) < 0 {
		dir = 0
	} else {
		if root.Link[0] == nil || root.Link[1] == nil {
			var child *Node[T]
			if root.Link[0] == nil {
				child = root.Link[1]
			} else {
				child = root.Link[0]
			}
			return child
		}

		heir := root.Link[1]
		for heir.Link[0] != nil {
			heir = heir.Link[0]
		}
		root.Data = heir.Data
		data = heir.Data
		dir = 1
	}

	updatedNode := tree.removeRecursive(root.Link[dir], data)
	root.Link[dir] = updatedNode
	if updatedNode == nil {
		return root
	}

	root.Balance += 1 - 2*dir
	switch root.Balance {
	case 0:
		return root
	case 1, -1:
		return root
	}

	return tree.removeBalance(root, 1 - dir)
}

func (tree *AVLTree[T]) removeBalance(root *Node[T], dir int) *Node[T] {
	n := root.Link[dir^1]
	bal := 2*(dir^1) - 1
	if n.Balance == -bal {
		root.Balance = 0
		n.Balance = 0
		return tree.singleRotate(root, dir)
	}
	if n.Balance == bal {
		tree.adjustBalance(root, dir^1, -bal)
	} else {
		root.Balance = -bal
		n.Balance = bal
	}
	return tree.doubleRotate(root, dir)
}

func (tree *AVLTree[T]) Remove(data T) {
	tree.Root = tree.removeRecursive(tree.Root, data)
}

func (tree *AVLTree[T]) Contains(data T) bool {
	current := tree.Root
	for current != nil {
		c := tree.comp(current.Data, data)
		if c == 0 {
			return true
		}
		if c < 0 {
			current = current.Link[1]
		} else {
			current = current.Link[0]
		}
	}
	return false
}

func (tree *AVLTree[T]) Successor(data T) *T {
	var successor *Node[T]
	current := tree.Root
	for current != nil {
		c1 := tree.comp(current.Data, data)
		if c1 > 0 {
			if successor == nil {
				successor = current
			}
			c2 := tree.comp(successor.Data, current.Data)
			if c2 > 0 {
				successor = current
			}
			current = current.Link[0]
		} else {
			current = current.Link[1]
		}
	}
	if successor != nil {
		return &successor.Data
	}
	return nil
}

func (tree *AVLTree[T]) Predecessor(data T) *T {
	var predecessor *Node[T]
	current := tree.Root
	for current != nil {
		c1 := tree.comp(current.Data, data)
		if c1 < 0 {
			if predecessor == nil {
				predecessor = current
			}
			c2 := tree.comp(predecessor.Data, current.Data)
			if c2 < 0 {
				predecessor = current
			}
			current = current.Link[1]
		} else {
			current = current.Link[0]
		}
	}
	if predecessor != nil {
		return &predecessor.Data
	}
	return nil
}
