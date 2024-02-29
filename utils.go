package main

import (
	"fmt"
	"reflect"
	"runtime"

	"golang.org/x/exp/constraints"
)

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func len64(slice interface{}) int64 {
	val := reflect.ValueOf(slice)
	return int64(val.Len())
}

func assert(cond bool) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("Assertion violation at %s:%d", file, line))
	}
}