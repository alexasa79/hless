package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/alexasa79/hless/internal/hhlog"
	"golang.org/x/exp/constraints"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var UL = hhlog.NewLogger("UTILS")
var assertHandlers = make([]func(), 0)
var timestampPrefixRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})[T ](\d{2}[-:]\d{2}[-:]\d{2}(?:\.\d+)?)(Z|[+-]\d{2}:?\d{2})?.*`)

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
		text := fmt.Sprintf("Assertion violation at %s:%d, meta=\n%s", file, line, meta)
		UL.Warning("%s", text)
		for _, f := range assertHandlers {
			f()
		}
		panic(text)
	}
}

func AddAssertHandler(f func()) {
	assertHandlers = append(assertHandlers, f)
}

func Check(err error) {
	if err != nil {
		panic(err)
	}
}

func ExtractTimestampPrefixPrecision(input string) (*time.Time, int, int) {
	matches := timestampPrefixRegex.FindStringSubmatch(input)
	if matches == nil || len(matches) != 4 {
		return nil, 0, 0
	}

	ts := matches[1] + "T" + strings.ReplaceAll(matches[2], "-", ":")
	tz := strings.ReplaceAll(matches[3], "Z", "+")
	if tz != "" && tz != "+" {
		if len(tz) == 5 {
			tz = tz[:3] + ":" + tz[3:]
		}
		ts += tz
	} else {
		ts += "+00:00"
	}

	res, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		UL.Warning("Error parsing %s: %v", ts, err)
	}

	precision := 0
	if dot := strings.Index(matches[2], "."); dot != -1 {
		precision = len(matches[2]) - dot - 1
	}

	return &res, len(matches[1]) + 1 + len(matches[2]) + len(matches[3]), precision
}

func ExtractTimestampPrefix(input string) (*time.Time, int) {
	res, end, _ := ExtractTimestampPrefixPrecision(input)
	return res, end
}

func ExtractTimestamp(input string) *time.Time {
	res, _ := ExtractTimestampPrefix(input)
	return res
}

type Word struct {
	Word  string
	Start int
	End   int
}

/* Split a given string into words and return their indexes in the original string */
func SplitWords(s string) []Word {
	words := make([]Word, 0)
	prev := true
	counter := 0
	fields := strings.FieldsFunc(s, func(r rune) bool {
		delim := unicode.IsSpace(r) || unicode.IsPunct(r) || r == '='
		if !delim && prev {
			words = append(words, Word{
				Start: counter,
			})
		}
		counter++
		prev = delim
		return delim
	})
	for i, f := range fields {
		words[i].Word = f
		words[i].End = words[i].Start + len(f)
	}
	return words
}

func CountLines(s string) int {
	return strings.Count(s, "\n")
}
