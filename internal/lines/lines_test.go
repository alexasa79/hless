package lines

import (
	"os"
	"testing"

	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
)

func checkLine(lines *Lines, line *Line, offset int64, length int64, rawLength int64) {
	utils.Assert(line != nil)
	utils.Assert(line.Offset == offset)
	utils.Assert(lines.Length(line) == length)
	utils.Assert(lines.RawLength(line) == rawLength)
}

func TestMain(m *testing.M) {
	hhlog.SetupLogging("../../logs/lines-test.log")
	defer hhlog.CleanupLogging()

	for _, logger := range hhlog.AllLoggers {
		logger.MinLevel = hhlog.TRACE
	}

	m.Run()
}

func TestLines1(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 16))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	utils.Assert(first != nil)
	utils.Assert(lines.Length(first) == 9)
	utils.Assert(lines.RawLength(first) == 10)

	utils.Assert(lines.Next(first) == nil)
}

func TestLines2(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 16))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n123456789\n"))
	<-lines.data.BufferRefreshChannel

	lines.readForward()
	utils.Assert(lines.currentSection.lines == 1)

	lines.readForward()
	utils.Assert(lines.currentSection.lines == 2)

	first := lines.First()
	utils.Assert(first != nil)
	utils.Assert(first.Offset == 0)
	utils.Assert(lines.Length(first) == 9)
	utils.Assert(lines.RawLength(first) == 10)

	second := lines.Next(first)
	utils.Assert(second != nil)
	utils.Assert(second.Offset == 10)
	utils.Assert(lines.Length(second) == 9)
	utils.Assert(lines.RawLength(second) == 10)

	utils.Assert(lines.Next(second) == nil)
}

func TestLinesAfterBlankLine(t *testing.T) {
	t.Parallel()
	lines := NewLines(NewBufferFromString("before\n\n==================== UPLOAD\n"))
	lines.batchSize = 1

	first := lines.First()
	checkLine(lines, first, 0, 6, 7)
	utils.Assert(lines.Text(first) == "before")

	blank := lines.Next(first)
	checkLine(lines, blank, 7, 0, 1)
	utils.Assert(lines.Text(blank) == "")

	upload := lines.Next(blank)
	checkLine(lines, upload, 8, 27, 28)
	utils.Assert(lines.Text(upload) == "==================== UPLOAD")
}

func TestLines3(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 16))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n123456789\n123456789\n"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	utils.Assert(first != nil)
	utils.Assert(first.Offset == 0)

	lines.readForward()
	lines.jumpTo(15, "@15")

	second := lines.Next(first)
	utils.Assert(second != nil)
	utils.Assert(second.Offset == 10)

	third := lines.Next(second)
	utils.Assert(third != nil)
	utils.Assert(third.Offset == 20)
}

func TestLines4(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 16))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n12345"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	utils.Assert(first != nil)
	utils.Assert(first.Offset == 0)

	second := lines.Next(first)
	utils.Assert(second != nil)
	utils.Assert(second.Offset == 10)
	utils.Assert(lines.Length(second) == 5)
	utils.Assert(lines.RawLength(second) == 5)

	w.Write([]byte("6789\n123456789\n"))
	<-lines.data.BufferRefreshChannel

	secondSecond := lines.Next(first)
	utils.Assert(second == secondSecond)
	utils.Assert(second.Offset == 10)
	utils.Assert(lines.Length(second) == 9)
	utils.Assert(lines.RawLength(second) == 10)

	third := lines.Next(second)
	checkLine(lines, third, 20, 9, 10)
}

func TestLines5(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 16))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("12345"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	checkLine(lines, first, 0, 5, 5)

	w.Write([]byte("6789\n12345"))
	<-lines.data.BufferRefreshChannel

	first = lines.First()
	checkLine(lines, first, 0, 9, 10)

	second := lines.Next(first)
	checkLine(lines, second, 10, 5, 5)
}

func TestLines6(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 100))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n123456789\n12345"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	checkLine(lines, first, 0, 9, 10)

	last := lines.Last()
	checkLine(lines, last, 20, 5, 5)

	utils.Assert(lines.sections.Count() == 2)

	w.Write([]byte("6789\n123456789\n"))
	<-lines.data.BufferRefreshChannel

	lastLast := lines.Last()
	utils.Assert(lastLast != last)
	checkLine(lines, lastLast, 30, 9, 10)

	third := lines.Prev(lastLast)
	checkLine(lines, third, 20, 9, 10)

	utils.Assert(lines.sections.Count() == 3)

	second := lines.Prev(third)
	checkLine(lines, second, 10, 9, 10)
}

func TestLines7(t *testing.T) {
	t.Parallel()
	r, w, _ := os.Pipe()
	lines := NewLines(NewBufferFromFileDescriptors(int(r.Fd()), 100))
	lines.batchSize = 1

	// Write 12 bytes
	w.Write([]byte("123456789\n123456789\n12345"))
	<-lines.data.BufferRefreshChannel

	first := lines.First()
	checkLine(lines, first, 0, 9, 10)

	last := lines.Last()
	checkLine(lines, last, 20, 5, 5)

	second := lines.Prev(last)
	checkLine(lines, second, 10, 9, 10)

	firstFirst := lines.Prev(second)
	utils.Assert(firstFirst == first)
	checkLine(lines, firstFirst, 0, 9, 10)
}
