package main

import (
	"fmt"
	"runtime"
)

func assert(cond bool) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("Assertion violation at %s:%d", file, line))
	}
}