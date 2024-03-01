package avltree

/// Returns -1 of a < b, 0 if a == b and 1 if a > b
type Comparator[T any] func(a, b T) int

type node[T any] struct {
	data T
	left *node[T]
	right *node[T]
	height int
}

type AVLTree[T any] struct {
	root *node[T]
	comp Comparator[T]
	count int64
}

func (n *node[T]) updateHeight() {
	left, right := -1, -1
	if n.left != nil {
		left = n.left.height
	}
	if n.right != nil {
		right = n.right.height
	}
	if left > right {
		n.height = left + 1
	} else {
		n.height = right + 1
	}
}

func (n *node[T]) rightRotation() *node[T] {
	n1 := n.left
	n.left = n1.right
	n1.right = n
	n.updateHeight()
	n1.updateHeight()
	return n1
}

func (n *node[T]) leftRotation() *node[T] {
	n1 := n.right
	n.right = n1.left
	n1.left = n
	n.updateHeight()
	n1.updateHeight()
	return n1
}

func (n *node[T]) balanceFactor() int {
	left, right := -1, -1
	if n.left != nil {
		left = n.left.height
	}
	if n.right != nil {
		right = n.right.height
	}
	return left - right
}

func (n *node[T]) rebalance() *node[T] {
	bf := n.balanceFactor()
	if bf > 1 {
		if n.left != nil && n.left.balanceFactor() < 0 {
			n.left = n.left.leftRotation()
		}
		n = n.rightRotation()
	} else if bf < -1 {
		if n.right != nil && n.right.balanceFactor() > 0 {
			n.right = n.right.rightRotation()
		}
		n = n.leftRotation()
	}
	return n
}

func (t *AVLTree[T]) insert(n, m *node[T]) *node[T] {
	if n == nil {
		t.count++
		return m
	}

	c := t.comp(m.data, n.data)
	if c < 0 {
		n.left = t.insert(n.left, m)
	} else if c > 0 {
		n.right = t.insert(n.right, m)
	} else {
		n.data = m.data
	}
	n.updateHeight()
	return n.rebalance()
}

func (t *AVLTree[T]) Insert(data T) {
	n := &node[T]{ data: data }
	t.root = t.insert(t.root, n)
}

func (t *AVLTree[T]) remove(n *node[T], data T) *node[T] {
    if n == nil {
        return nil
    }
	c := t.comp(data, n.data)

    if c < 0 {
        n.left = t.remove(n.left, data)
    } else if c > 0 {
        n.right = t.remove(n.right, data)
    } else {
		t.count--
        if n.right != nil {
            successor := n.right
            for successor.left != nil {
                successor = successor.left
            }
            n.data = successor.data
            n.right = t.remove(n.right, successor.data)
        } else if n.left != nil {
            n = n.left
        } else {
            n = nil
            return n
        }
    }
    if n != nil {
        n.updateHeight()
        n = n.rebalance()
    }
    return n
}

func (t *AVLTree[T]) Remove(data T) {
	t.root = t.remove(t.root, data)
}

func (t *AVLTree[T]) Contains(data T) bool {
	for current := t.root; current != nil; {
		c := t.comp(data, current.data)
		if c < 0 {
			current = current.left
		} else if c > 0 {
			current = current.right
		} else {
			return true
		}
	}
	return false
}

func (t *AVLTree[T]) Get(data T) *T {
	for current := t.root; current != nil; {
		c := t.comp(data, current.data)
		if c < 0 {
			current = current.left
		} else if c > 0 {
			current = current.right
		} else {
			return &current.data
		}
	}
	return nil
}

func (t *AVLTree[T]) Successor(data T) *T {
	var successor *node[T]
	for current := t.root; current != nil; {
		c1 := t.comp(data, current.data)
		if c1 < 0 {
			if successor == nil {
				successor = current
			}
			c2 := t.comp(successor.data, current.data)
			if c2 > 0 {
				successor = current
			}
			current = current.left
		} else {
			current = current.right
		}
	}
	if successor != nil {
		return &successor.data
	}
	return nil
}

func (tree *AVLTree[T]) Predecessor(data T) *T {
	var predecessor *node[T]
	current := tree.root
	for current != nil {
		c1 := tree.comp(data, current.data)
		if c1 > 0 {
			if predecessor == nil {
				predecessor = current
			}
			c2 := tree.comp(predecessor.data, current.data)
			if c2 < 0 {
				predecessor = current
			}
			current = current.right
		} else {
			current = current.left
		}
	}
	if predecessor != nil {
		return &predecessor.data
	}
	return nil
}

func (tree *AVLTree[T]) Count() int64 {
	return tree.count
}

func NewAVLTree[T any](comp Comparator[T]) *AVLTree[T] {
	return &AVLTree[T]{ nil, comp, 0 }
}
