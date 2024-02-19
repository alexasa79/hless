package main

import (
	"strings"
	"testing"
)

func createModel(s string, w int, h int) *Model {
	m := initialModel(strings.NewReader(s))
	m.pagerWidth = w
	m.pagerHeight = h
	m.ready = true
	return m
}

func TestScrollDown(t *testing.T) {
	m := createModel("this is a test string", 2, 2)
	m.OnKeyDown()
	if m.top.Line != 0 || m.top.Segment != 1 {
		t.Fatalf("Unexpected top %v", m.top)
	}
}
