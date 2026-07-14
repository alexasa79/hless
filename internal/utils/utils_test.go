package utils

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		want          string
		wantEnd       int
		wantPrecision int
	}{
		{
			name:          "space-separated nanoseconds",
			input:         "2026-06-22 18:22:39.586770174 message",
			want:          "2026-06-22T18:22:39.586770174Z",
			wantEnd:       len("2026-06-22 18:22:39.586770174"),
			wantPrecision: 9,
		},
		{
			name:          "space-separated microseconds",
			input:         "2026-06-22 18:22:39.586770 message",
			want:          "2026-06-22T18:22:39.586770Z",
			wantEnd:       len("2026-06-22 18:22:39.586770"),
			wantPrecision: 6,
		},
		{
			name:          "space-separated milliseconds",
			input:         "2026-06-22 18:22:39.586 message",
			want:          "2026-06-22T18:22:39.586Z",
			wantEnd:       len("2026-06-22 18:22:39.586"),
			wantPrecision: 3,
		},
		{
			name:          "rfc3339 nanoseconds",
			input:         "2026-06-22T18:22:39.586770174 message",
			want:          "2026-06-22T18:22:39.586770174Z",
			wantEnd:       len("2026-06-22T18:22:39.586770174"),
			wantPrecision: 9,
		},
		{
			name:          "hyphen-separated time",
			input:         "2026-06-22T18-22-39.586770174 message",
			want:          "2026-06-22T18:22:39.586770174Z",
			wantEnd:       len("2026-06-22T18-22-39.586770174"),
			wantPrecision: 9,
		},
		{
			name:          "zulu timezone",
			input:         "2026-06-22T18:22:39.586770174Z message",
			want:          "2026-06-22T18:22:39.586770174Z",
			wantEnd:       len("2026-06-22T18:22:39.586770174Z"),
			wantPrecision: 9,
		},
		{
			name:          "bracketed zulu timestamp",
			input:         "[2026-07-14T13:31:57Z] message",
			want:          "2026-07-14T13:31:57Z",
			wantEnd:       len("[2026-07-14T13:31:57Z]"),
			wantPrecision: 0,
		},
		{
			name:          "colon timezone",
			input:         "2026-06-22T18:22:39.586770174+03:30 message",
			want:          "2026-06-22T18:22:39.586770174+03:30",
			wantEnd:       len("2026-06-22T18:22:39.586770174+03:30"),
			wantPrecision: 9,
		},
		{
			name:          "compact timezone",
			input:         "2026-06-22T18:22:39.586770174+0330 message",
			want:          "2026-06-22T18:22:39.586770174+03:30",
			wantEnd:       len("2026-06-22T18:22:39.586770174+0330"),
			wantPrecision: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotEnd, gotPrecision := ExtractTimestampPrefixPrecision(tt.input)
			if got == nil {
				t.Fatalf("ExtractTimestamp(%q) returned nil", tt.input)
			}
			if gotEnd != tt.wantEnd {
				t.Fatalf("timestamp end = %d, want %d", gotEnd, tt.wantEnd)
			}
			if gotPrecision != tt.wantPrecision {
				t.Fatalf("timestamp precision = %d, want %d", gotPrecision, tt.wantPrecision)
			}

			want, err := time.Parse(time.RFC3339Nano, tt.want)
			if err != nil {
				t.Fatal(err)
			}

			if !got.Equal(want) {
				t.Fatalf("ExtractTimestamp(%q) = %s, want %s", tt.input, got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
			}
		})
	}
}

func TestExtractTimestampNoMatch(t *testing.T) {
	t.Parallel()

	if got := ExtractTimestamp("not a timestamp"); got != nil {
		t.Fatalf("ExtractTimestamp returned %s, want nil", got.Format(time.RFC3339Nano))
	}
}

func TestSplitWordsTreatsEqualsAsDelimiter(t *testing.T) {
	t.Parallel()

	got := SplitWords("rid=0x1243")
	want := []Word{
		{Word: "rid", Start: 0, End: 3},
		{Word: "0x1243", Start: 4, End: 10},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitWords() = %#v, want %#v", got, want)
	}
}
