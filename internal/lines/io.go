package lines

import (
	"fmt"
	"math"

	"github.com/alexasa79/hless/internal/utils"
)

func (lines *Lines) readForward() {
	s := lines.currentSection
	LL.Debug("Reading forwards in section %v from offset %d, size %d",
		s, s.read.end, lines.data.Size())

	if s.read.end >= lines.data.Size() {
		return
	}

	batchSize := 256

	lineStart := s.read.end
	pos := s.read.end
	for pos < lines.data.Size() {
		if lines.data.At(pos) == '\n' {
			len := pos - lineStart
			line := Line{
				Offset:    lineStart,
				Length:    len,
				index:     s.nextIndexForward,
				RawLength: pos - lineStart + 1,
				section:   lines.currentSection,
			}

			lines.lines.Insert(&line)
			LL.Trace("Inserted line %v", line)

			s.lines++
			s.nextIndexForward++

			pos++
			lineStart = pos

			batchSize--
			if batchSize == 0 {
				break
			}
		} else {
			pos++
		}
	}
	s.read.end = pos
}

func (lines *Lines) readBackward() {
	s := lines.currentSection
	LL.Debug("Reading backwards in section %v", s)

	if s.read.start == 0 {
		return
	}

	// s.read.start points at the first character in the last line found string.
	// Decrementing pos by 1 will get us to point at the \n. Decrementing once again will
	// get us to look at the last character in the line we're reading.
	pos := s.read.start - 2
	for pos >= 0 {
		if lines.data.At(pos) == '\n' {
			len := s.read.start - pos - 2
			line := Line{
				Offset:    pos + 1,
				Length:    len,
				index:     s.nextIndexBackward,
				RawLength: s.read.start - pos - 1,
				section:   lines.currentSection,
			}

			LL.Trace("Inserted line %v", line)
			lines.lines.Insert(&line)

			s.lines++
			s.nextIndexBackward--

			break
		}
		pos--
	}

	if pos < 0 {
		len := s.read.start - 1
		if lines.data.At(len-1) == '\r' {
			len--
		}

		line := Line{
			Offset:    0,
			Length:    len,
			index:     s.nextIndexBackward,
			RawLength: s.read.start,
			section:   lines.currentSection,
		}

		lines.lines.Insert(&line)

		s.lines++
		s.nextIndexBackward--
	}

	s.read.start = pos + 1
}

func (lines *Lines) first(iteration int) *Line {
	utils.Assert(iteration < 100000)

	LL.Trace("Asked to return first in the buffer. Iteration %d", iteration)

	if lines.data.fileSize == 0 {
		return nil
	}

	lines.jumpTo(0, "")
	t := lines.lines.Get(&Line{Offset: 0})
	if t != nil {
		return *t
	} else {
		lines.readForward()
		return lines.first(iteration + 1)
	}
}

func (lines *Lines) last(iteration int) *Line {
	utils.Assert(iteration < 100000)

	LL.Trace("Asked to return last in the buffer. Iteration %d", iteration)

	size := lines.data.Size()
	lines.jumpTo(size, fmt.Sprintf("@%d", size))
	t := lines.lines.Predecessor(&Line{Offset: math.MaxInt64})
	if t != nil {
		LL.Trace("Predecessor of infinity is %v", *t)
	}

	if t != nil && (*t).Offset+(*t).RawLength >= size {
		return *t
	} else {
		lines.readBackward()
		return lines.last(iteration + 1)
	}
}

func (lines *Lines) next(line *Line, iterations int) *Line {
	LL.Trace("Asked to return next of %v. Data size %d. Iteration %d", line, lines.data.Size(), iterations)

	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	utils.Assert(iterations < 100000)

	if line.Offset+int64(line.RawLength) >= lines.data.Size() {
		return nil
	}

	var next *Line = nil
	t := lines.lines.Successor(line)
	if t != nil {
		next = *t
	}

	if next == nil {
		lines.readForward()
		return lines.next(line, iterations+1)
	} else {
		if line.Offset+int64(line.RawLength) == int64(next.Offset) {
			return next
		} else {
			lines.readForward()
			return lines.next(line, iterations+1)
		}
	}
}

func (lines *Lines) prev(line *Line, iterations int) *Line {
	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	utils.Assert(iterations < 100000)

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
		return lines.prev(line, iterations+1)
	} else {
		if prev.Offset+int64(prev.RawLength) == int64(line.Offset) {
			return prev
		} else {
			lines.readBackward()
			return lines.prev(line, iterations+1)
		}
	}
}

func (lines *Lines) jumpTo(offset int64, name string) {
	s := Section{read: Range{start: offset, end: offset}}
	t := lines.sections.Get(&s)
	if t != nil {
		lines.currentSection = *t
		return
	}

	t = lines.sections.Predecessor(&s)
	if t != nil {
		pred := *t
		if offset >= pred.read.start && offset < pred.read.end {
			lines.currentSection = pred
			return
		} else if offset == pred.read.end {
			lines.currentSection = pred
			return
		}
	}

	section := newSection(offset, name)

	lines.currentSection = section
	lines.sections.Insert(section)
}

func (lines *Lines) at(offset int64) *Line {
	name := fmt.Sprintf("@%d", offset)
	lines.jumpTo(offset, name)

	s := lines.currentSection

	t := lines.lines.Predecessor(&Line{Offset: offset})
	if t != nil && (*t).section == s {
		return *t
	}

	// Read back until we run into end of line. Then read forward...
	pos := offset
	for pos >= 0 {
		if lines.data.At(pos) == '\n' {
			pos++
			break
		}
		pos--
	}

	// Making sure that next line we add in readForward will get index 0. Forward index
	// is set to 1 by default to accomodate the first section. This does not work for
	// sections we add in the middle of the file.
	s.nextIndexForward = 0
	s.nextIndexBackward = -1

	s.read = Range{pos, pos}
	lines.readForward()

	t = lines.lines.Get(&Line{Offset: pos})
	utils.Assert(t != nil, offset, name)

	return *t
}

func (lines *Lines) text(line *Line) string {
	return string(lines.data.Range(line.Offset, line.Offset+line.Length))
}
