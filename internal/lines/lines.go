package lines

import (
	"fmt"
	"io"
	"math"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
)

var LL *hhlog.Logger

func SectionComp(a, b *Section) int {
	if a.offset < b.offset {
		return -1
	} else if a.offset == b.offset {
		return 0
	} else {
		return 1
	}
}

type Section struct {
	offset int64
	name string

	data []byte
	parsedStart int // first parsed byte
	parsedEnd int // last parsed byte
	lines int // how many lines we found

	nextIndexForward int
	nextIndexBackward int
}

func LineComp(a, b *Line) int {
	if a.Offset < b.Offset {
		return -1
	} else if a.Offset == b.Offset {
		return 0
	} else {
		return 1
	}
}

type Line struct {
	Offset int64
	len int // length of text
	index int // index in section
	rawLength int // raw length, with end of line
	section *Section
}

type Lines struct {
	file io.ReadSeeker
	fileSize int64
	eof bool

	batchSize int

	lines *avltree.AVLTree[* Line]
	sections *avltree.AVLTree[* Section]

	currentSection *Section
}

func newSection(offset int64, name string) *Section {
	return &Section{
		data: make([]byte, 0),
		offset: offset,
		name: name,
		nextIndexForward: 1,
		nextIndexBackward: -1,
	}
}

func NewLines(file io.ReadSeeker, fileSize int64) *Lines {
	lines := &Lines{
		file: file,
		fileSize: fileSize,
		eof: false,

		batchSize: 65536,

		lines: avltree.NewAVLTree[* Line](LineComp),
		sections: avltree.NewAVLTree[*Section](SectionComp),

		currentSection: newSection(0, ""),
	}

	lines.sections.Insert(lines.currentSection)

	LL.Debug("Started new section %v", lines.currentSection)

	return lines
}

func (lines Line) String() string {
	return fmt.Sprintf("[Line: @%d, len=%d, rlen=%d, index=%d, sec=%p]",
		lines.Offset, lines.len, lines.rawLength, lines.index, lines.section)
}

func (s Section) String() string {
	return fmt.Sprintf("[Section: @%p, offset=%d, name=%s, lines=%d, buf=%d, parsed=%d,%d, index=%d,%d]",
		&s, s.offset, s.name, s.lines, len(s.data), s.parsedStart, s.parsedEnd,
		s.nextIndexForward, s.nextIndexBackward)

}

func (lines *Lines) FileSize() int64 {
	return lines.fileSize
}

func (lines *Lines) readForward() {
	buf := make([]byte, lines.batchSize)

	s := lines.currentSection
	lines.file.Seek(s.offset + utils.Len64(s.data), io.SeekStart)

	LL.Debug("Reading forward in section %v from offset %d", s, s.offset + utils.Len64(s.data))

	n, err := lines.file.Read(buf)
	if err != nil {
		return
	}

	LL.Debug("Read %d bytes", n)

	s.data = append(s.data, buf[:n]...)
	for i := s.parsedEnd; i < len(s.data); i++ {
		if s.data[i] == '\n' { // 123\n
			len := i - s.parsedEnd
			if i > 0 && s.data[i - 1] == '\r' {
				len--
			}

			line := Line {
				Offset: int64(s.parsedEnd) + s.offset,
				len: len,
				index: s.nextIndexForward,
				rawLength: i - s.parsedEnd + 1,
				section: lines.currentSection,
			}

			lines.lines.Insert(&line)

			s.lines++
			s.nextIndexForward++
			s.parsedEnd = i + 1

			LL.Trace("Added line %v", line)
		}
	}

	LL.Debug("Section after reading %v", s)
}

func (lines *Lines) readBackward() {
	s := lines.currentSection
	if s.offset == 0 && s.parsedStart == 0 {
		return
	}

	batchSize := utils.Min(int64(lines.batchSize), s.offset)
	s.offset -= batchSize
	lines.file.Seek(s.offset, io.SeekStart)

	LL.Debug("Reading backward in section %v from offset %d", s, s.offset)

	buf := make([]byte, batchSize)
	n, err := lines.file.Read(buf)
	if err != nil {
		return
	}

	LL.Debug("Read %d bytes", n)

	s.data = append(buf[:n], s.data...)
	s.parsedStart += n
	s.parsedEnd += n
	for i := s.parsedStart - 1; i >= 0; i-- {
		len := s.parsedStart - i - 1
		if s.data[i] == '\n' && len > 0 {
			if s.data[i + len - 1] == '\r' {
				len--
			}

			line := Line {
				Offset: int64(i) + s.offset + 1,
				len: len - 1,
				index: s.nextIndexBackward,
				rawLength: s.parsedStart - i - 1,
				section: lines.currentSection,
			}

			lines.lines.Insert(&line)

			s.lines++
			s.nextIndexBackward--
			s.parsedStart = i + 1

			LL.Trace("Added line %v", line)
		}
	}

	if s.offset == 0 {
		len := s.parsedStart - 1
		if s.data[len - 1] == '\r' {
			len--
		}

		line := Line {
			Offset: 0,
			len: len,
			index: s.nextIndexBackward,
			rawLength: s.parsedStart,
			section: lines.currentSection,
		}

		lines.lines.Insert(&line)

		s.lines++
		s.nextIndexBackward--
		s.parsedStart = 0

		LL.Trace("Added line %v", line)
	}

	LL.Debug("Section after reading %v", s)
}

func (lines *Lines) First() *Line {
	lines.JumpTo(0, "")
	lines.readForward()
	t := lines.lines.Get(&Line{ Offset: 0 })
	if t != nil {
		return *t
	}
	return nil
}

func (lines *Lines) Last() *Line {
	lines.JumpTo(lines.fileSize, "end")
	lines.readBackward()
	t := lines.lines.Predecessor(&Line{ Offset: math.MaxInt64 })
	if t != nil {
		return *t
	}
	return nil
}

func (lines *Lines) moveToSection(line *Line, section *Section) {
	utils.Assert(line.section != section)

	// Works only if the line is at the beginning of its section or at the end of its section.
	s2 := line.section
	utils.Assert(int64(s2.parsedStart) + s2.offset == line.Offset ||
		int64(s2.parsedEnd) + s2.offset == line.Offset + int64(line.rawLength),
		section, line, s2)

	if int64(s2.parsedStart) + s2.offset == line.Offset {
		// The line is the first in its section. We have to trim the section's buffer and
		// (parsedStart, parsedEnd)
	}
}

func (lines *Lines) next(line *Line, iterations int) *Line {
	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	if iterations > 100 {
		return nil
	}

	if line.Offset + int64(line.rawLength) >= lines.fileSize {
		return nil
	}

	var next *Line = nil
	t := lines.lines.Successor(line)
	if t == nil && lines.eof {
		return nil
	}

	if t != nil {
		next = *t
	}

	if next == nil {
		lines.readForward()
		return lines.next(line, iterations + 1)
	} else {
		if line.Offset + int64(line.rawLength) == int64(next.Offset) {
			if line.section != next.section {
				lines.moveToSection(next, line.section)
			}

			return next
		} else {
			lines.readForward()
			return lines.next(line, iterations + 1)
		}
	}
}

func (lines *Lines) Next(line *Line) (next *Line) {
	next = lines.next(line, 0)
	LL.Trace("Asked for next of %v, Got %v", line, next)
	return next
}

func (lines *Lines) prev(line *Line, iterations int) *Line {
	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	if iterations > 100 {
		return nil
	}

	if line.Offset == 0 {
		return nil
	}

	var prev *Line = nil
	t := lines.lines.Predecessor(line)
	if t != nil {
		prev = *t
	}

	if prev == nil {
		lines.readBackward()
		return lines.prev(line, iterations + 1)
	} else {
		if prev.Offset + int64(prev.rawLength) == int64(line.Offset) {
			if line.section != prev.section {
				lines.moveToSection(line, prev.section)
			}
			return prev
		} else {
			lines.readBackward()
			return lines.prev(line, iterations + 1)
		}
	}
}

func (lines *Lines) Prev(line *Line) (prev *Line) {
	prev = lines.prev(line, 0)
	LL.Trace("Asked for prev of %v, Got %v", line, prev)
	return prev
}

func (lines *Lines) JumpTo(offset int64, name string) {
	s := Section{ offset: offset }
	t := lines.sections.Get(&s)
	if t != nil {
		lines.currentSection = *t
		return
	}

	t = lines.sections.Predecessor(&s)
	if t != nil {
		pred := *t
		if offset >= pred.offset && offset <= pred.offset + utils.Len64(pred.data) {
			lines.currentSection = pred
			return
		}
	}

	section := newSection(offset, name)

	lines.currentSection = section
	lines.sections.Insert(section)

	LL.Debug("Started new section %v", section)
}

func (lines *Line) Text() string {
	s := lines.section
	start := int(lines.Offset - s.offset)
	end := start + lines.len
	return string(s.data[start:end])
}

func (lines *Line) Name() string {
	s := lines.section
	if s.name == "" {
		return fmt.Sprintf("%d", lines.index)
	}
	t := fmt.Sprintf("%s%+d", s.name, lines.index)
	return t
}

func Distance(l1, l2 *Line) string {
	if l1.section == l2.section {
		dist := l2.index - l1.index
		if dist < 0 {
			dist *= -1
		}
		return fmt.Sprintf("%d", dist)
	} else {
		return "⋯" + l1.Name()
	}
}