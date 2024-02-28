package main

import (
	"fmt"
	"runtime"
)

func min(a, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}

func assert(cond bool) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("Assertion violation at %s:%d", file, line))
	}
}