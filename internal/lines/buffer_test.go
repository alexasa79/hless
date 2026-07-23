package lines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexasa79/hless/internal/utils"
)

func TestNewBufferFromEmptyFile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(filename, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	buf := NewBufferFromFile(filename)

	if got := buf.Size(); got != 0 {
		t.Fatalf("Size() = %d, want 0", got)
	}
	if got := buf.Range(0, 0); len(got) != 0 {
		t.Fatalf("Range(0, 0) returned %d bytes, want 0", len(got))
	}
}

func TestFileBufferRefreshesBetweenEmptyAndNonEmpty(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filename, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	buf := NewBufferFromFile(filename)

	if err := os.WriteFile(filename, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed := buf.Refresh(); !changed {
		t.Fatal("Refresh() = false after file became non-empty")
	}
	if got := string(buf.Range(0, buf.Size())); got != "hello" {
		t.Fatalf("buffer contents = %q, want %q", got, "hello")
	}

	if err := os.WriteFile(filename, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if changed := buf.Refresh(); !changed {
		t.Fatal("Refresh() = false after file became empty")
	}
	if got := buf.Size(); got != 0 {
		t.Fatalf("Size() = %d, want 0", got)
	}
}

func TestBufferExtractsLongAndUnterminatedLines(t *testing.T) {
	long := strings.Repeat("x", 512*1024)
	buf := NewBufferFromString(long + "\nshort\nunterminated")

	if got := buf.lineText(0); got != long {
		t.Fatalf("lineText(0) returned %d bytes, want %d", len(got), len(long))
	}
	if length, rawLength := buf.lineLengths(0); length != int64(len(long)) || rawLength != int64(len(long)+1) {
		t.Fatalf("lineLengths(0) = (%d, %d), want (%d, %d)", length, rawLength, len(long), len(long)+1)
	}

	unterminatedOffset := int64(len(long) + len("\nshort\n"))
	if got := buf.lineText(unterminatedOffset); got != "unterminated" {
		t.Fatalf("lineText(%d) = %q, want %q", unterminatedOffset, got, "unterminated")
	}
	if length, rawLength := buf.lineLengths(unterminatedOffset); length != 12 || rawLength != 12 {
		t.Fatalf("lineLengths(%d) = (%d, %d), want (12, 12)", unterminatedOffset, length, rawLength)
	}
}

func TestReadFileDoesNotHoldBufferLockWhileRefreshChannelIsFull(t *testing.T) {
	t.Parallel()

	r, w, err := os.Pipe()
	utils.Assert(err == nil, err)
	defer r.Close()
	defer w.Close()

	buf := NewBufferFromFileDescriptors(int(r.Fd()), 1)

	for i := 0; i < cap(buf.BufferRefreshChannel); i++ {
		w.Write([]byte("a"))
		<-buf.BufferRefreshChannel
	}

	for i := 0; i < cap(buf.BufferRefreshChannel); i++ {
		buf.BufferRefreshChannel <- BufferRefreshSignal(1)
	}

	w.Write([]byte("b"))

	deadline := time.After(time.Second)
	for {
		if buf.Size() == int64(cap(buf.BufferRefreshChannel)+1) {
			return
		}

		select {
		case <-deadline:
			t.Fatal("buffer lock remained blocked behind full refresh channel")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
