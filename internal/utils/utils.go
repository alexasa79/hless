package utils

import (
	"fmt"
	"reflect"
	"runtime"

	"golang.org/x/exp/constraints"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func Sprint64(n int64) string {
	p := message.NewPrinter(language.English)
    return p.Sprintf("%d", n)
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Len64(slice interface{}) int64 {
	val := reflect.ValueOf(slice)
	return int64(val.Len())
}

func Assert(cond bool) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("Assertion violation at %s:%d", file, line))
	}
}
