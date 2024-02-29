package main

import (
	"testing"
)

func TestIntTree1(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	for i := int64(0); i < 100; i++ {
		tree.Insert(i)
	}

	for i := int64(0); i < 100; i++ {
		if !tree.Contains(i) {
			t.Fatalf("Could not find element %d in the tree\n", i)
		}
	}
}

func TestIntSuccessor(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	for i := int64(0); i < 100; i += 10 {
		tree.Insert(i)
	}

	for i := int64(0); i < 100; i += 1 {
		s := tree.Successor(i)
		if s == nil && i < 90 {
			t.Fatalf("Unexpected successor of %d: %v\n", i, s)
		} else if s != nil && *s != (i / 10) * 10 + 10 {
			t.Fatalf("Unexpected successor of %d: %v\n", i, s)
		}
	}
}

func TestIntPredecessor(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	for i := int64(0); i < 100; i += 10 {
		tree.Insert(i)
	}

	for i := int64(0); i < 100; i += 1 {
		s := tree.Predecessor(i)
		if s == nil && i > 0 {
			t.Fatalf("Unexpected predecessor of %d: %v\n", i, *s)
		} else if s != nil && *s != ((i - 1) / 10) * 10 {
			t.Fatalf("Unexpected predecessor of %d: %v\n", i, *s)
		}
	}
}

func TestContains(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	tree.Insert(10)
	if !tree.Contains(10) {
		t.Fatalf("Missing 10\n")
	}
	if tree.Contains(0) {
		t.Fatalf("Unexpected 0\n")
	}
	tree.Insert(0)
	if !tree.Contains(0) {
		t.Fatalf("Missing 10\n")
	}
}

func TestRemove(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	tree.Insert(0)
	tree.Remove(0)
	if tree.Contains(0) {
		t.Fatalf("Unexpected 0\n")
	}
}

func TestCount(t *testing.T) {
	tree := NewAVLTree(int64Comp)
	if tree.Count() != 0 {
		t.Fatalf("Unexpected number of elements")
	}
	tree.Insert(0)
	if tree.Count() != 1 {
		t.Fatalf("Unexpected number of elements")
	}
	tree.Insert(0)
	if tree.Count() != 1 {
		t.Fatalf("Unexpected number of elements")
	}
	tree.Remove(1)
	if tree.Count() != 1 {
		t.Fatalf("Unexpected number of elements")
	}
	tree.Remove(0)
	if tree.Count() != 0 {
		t.Fatalf("Unexpected number of elements")
	}
}

func TestSearchRanges(t *testing.T) {
	tree := NewAVLTree(searchRangeComp)
	tree.Insert(SearchRange{1, 1})
	tree.Insert(SearchRange{2, 2})
	tree.Remove(SearchRange{2, 2})
	tree.Insert(SearchRange{2, 3})
}
