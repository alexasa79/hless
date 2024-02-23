package main

import (
	"testing"
)

func TestIntTree1(t *testing.T) {
	tree := NewAVLTree[int](intComp)
	for i := 0; i < 100; i++ {
		tree.Insert(i)
	}

	for i := 0; i < 100; i++ {
		if !tree.Contains(i) {
			t.Fatalf("Could not find element %d in the tree\n", i)
		}
	}
}

func TestIntSuccessor(t *testing.T) {
	tree := NewAVLTree[int](intComp)
	for i := 0; i < 100; i += 10 {
		tree.Insert(i)
	}

	for i := 0; i < 100; i += 1 {
		s := tree.Successor(i)
		if s == nil && i < 90 {
			t.Fatalf("Unexpected successor of %d: %v\n", i, s)
		} else if s != nil && *s != (i / 10) * 10 + 10 {
			t.Fatalf("Unexpected successor of %d: %v\n", i, s)
		}
	}
}

func TestIntPredecessor(t *testing.T) {
	tree := NewAVLTree[int](intComp)
	for i := 0; i < 100; i += 10 {
		tree.Insert(i)
	}

	for i := 0; i < 100; i += 1 {
		s := tree.Predecessor(i)
		if s == nil && i > 0 {
			t.Fatalf("Unexpected predecessor of %d: %v\n", i, *s)
		} else if s != nil && *s != ((i - 1) / 10) * 10 {
			t.Fatalf("Unexpected predecessor of %d: %v\n", i, *s)
		}
	}
}

func TestContains(t *testing.T) {
	tree := NewAVLTree[int](intComp)
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
	tree := NewAVLTree[int](intComp)
	tree.Insert(0)
	tree.Remove(0)
	if tree.Contains(0) {
		t.Fatalf("Unexpected 10\n")
	}
}