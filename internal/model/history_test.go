package model

import (
	"reflect"
	"testing"
)

func TestHistoryAppendRecordsEveryEntry(t *testing.T) {
	t.Parallel()

	history := NewHistory()
	history.Append("first")
	history.Append("first")
	history.Append("second")

	want := []string{"first", "first", "second"}
	if !reflect.DeepEqual(history.History, want) {
		t.Fatalf("history = %#v, want %#v", history.History, want)
	}
}
