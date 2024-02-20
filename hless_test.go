package main

import (
	"strings"
	"testing"
)

func createModel(s string, w int, h int) *Model {
	m := initialModel(strings.NewReader(s))
	m.width = w
	m.height = h
	m.ready = true
	m.Init()
	return m
}

func (m *Model) verifyState(t *testing.T, line int, segment int, cursor int) {
	if m.top.Line != line || m.top.Segment != segment || m.cursorLine != cursor {
		t.Fatalf("Unexpected state: top=%v, cursor=%v. Expected %v and %v\n",
			m.top, m.cursorLine, Position{ line, segment }, cursor)
	}
}

func TestDown1(t *testing.T) {
	m := createModel("012345678\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown() // showing 12
	m.verifyState(t, 0, 1, 0)

	m.onKeyDown() // 23
	m.onKeyDown() // 34
	m.onKeyDown() // 45
	m.onKeyDown() // 56
	m.onKeyDown() // 67
	m.onKeyDown() // 78
	m.verifyState(t, 0, 7, 0)

	m.onKeyDown() // 78
	m.verifyState(t, 0, 7, 0)
}

func TestDown2(t *testing.T) {
	m := createModel("01\n23\n45\n67\n78\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown() // showing 1\n2
	m.verifyState(t, 0, 1, 1)
}

func TestUp(t *testing.T) {
	m := createModel("012345678\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown()
	m.verifyState(t, 0, 1, 0)
	m.onKeyUp()
	m.verifyState(t, 0, 0, 0)
}

func TestPageDown(t *testing.T) {
	m := createModel("01\n23\n45\n67\n78\n", 1, 2)
	m.onPageDown()
	m.verifyState(t, 0, 0, 0)
	m.onPageDown()
	m.verifyState(t, 0, 0, 0)
}
