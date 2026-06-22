package utils

import (
	"testing"
	"time"
)

func TestExtractTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "space-separated nanoseconds",
			input: "2026-06-22 18:22:39.586770174 message",
			want:  "2026-06-22T18:22:39.586770174Z",
		},
		{
			name:  "rfc3339 nanoseconds",
			input: "2026-06-22T18:22:39.586770174 message",
			want:  "2026-06-22T18:22:39.586770174Z",
		},
		{
			name:  "hyphen-separated time",
			input: "2026-06-22T18-22-39.586770174 message",
			want:  "2026-06-22T18:22:39.586770174Z",
		},
		{
			name:  "zulu timezone",
			input: "2026-06-22T18:22:39.586770174Z message",
			want:  "2026-06-22T18:22:39.586770174Z",
		},
		{
			name:  "colon timezone",
			input: "2026-06-22T18:22:39.586770174+03:30 message",
			want:  "2026-06-22T18:22:39.586770174+03:30",
		},
		{
			name:  "compact timezone",
			input: "2026-06-22T18:22:39.586770174+0330 message",
			want:  "2026-06-22T18:22:39.586770174+03:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTimestamp(tt.input)
			if got == nil {
				t.Fatalf("ExtractTimestamp(%q) returned nil", tt.input)
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
