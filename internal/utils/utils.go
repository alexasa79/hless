package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

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

func Assert(cond bool, args ...interface{}) {
	if !cond {
		_, file, line, _ := runtime.Caller(1)
		meta := ""
		for _, arg := range args {
			meta += fmt.Sprintf("%+v\n", arg)
		}
		panic(fmt.Sprintf("Assertion violation at %s:%d, meta=\n%s", file, line, meta))
	}
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func ExtractTimestamp(input string) (*time.Time) {
	regex := `^(\d{4}-\d{2}-\d{2})T(\d{2}[-:]\d{2}[-:]\d{2}\.\d+)?(Z|[+-]\d{2}:?\d{2})?.*`
	r := regexp.MustCompile(regex)

	matches := r.FindStringSubmatch(input)
	if matches == nil || len(matches) != 4 {
		return nil
	}

	ts := matches[1]
	ts += "T"
	ts += strings.ReplaceAll(matches[2], "-", ":")
	ts += strings.Repeat("0", 18 - len(matches[2]))
	tz := strings.ReplaceAll(matches[3], "Z", "+")
	if tz != "" && tz != "+" {
		ts += tz
	} else {
		ts += "+00:00"
	}

	res, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		fmt.Println("error:", err)
	}

	return &res
}
