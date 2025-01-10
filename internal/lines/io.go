package lines

import (
	"fmt"
	"math"

	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
)

var LIO *hhlog.Logger = hhlog.NewLogger("IO")

func (lines *Lines) readForward() {
	s := lines.currentSection
	LIO.Debug("Reading forwards in section %v from offset %d, size %d",
		s, s.read.end, lines.data.Size())

	lineStart := s.read.end
	pos := s.read.end
	first := true
	batchSize := lines.batchSize

	for pos < lines.data.Size() && batchSize > 0 {
		c := lines.data.At(pos)
		if c == '\n' || pos == lines.data.Size()-1 {
			line := Line{
				Offset:  lineStart,
				index:   s.nextIndexForward,
				section: lines.currentSection,
			}

			insert := true
			if first {
				predPtr := lines.lines.Predecessor(&line)
				if predPtr != nil {
					pred := *predPtr
					rawL := lines.RawLength(pred)
					if pred.Offset+rawL > lineStart {
						LIO.Trace("Line %v already exists. Skipping", pred)
						insert = false
					}
				}
				first = false
			}

			if insert {
				lines.lines.Insert(&line)
				LIO.Trace("Inserted line %v", &line)

				s.lines++
				s.nextIndexForward++
				batchSize--
			}

			pos++
			lineStart = pos
			s.read.end = pos

		}
		pos++
	}

	LIO.Debug("Section after read %v, size %d", s, lines.data.Size())
}

func (lines *Lines) readBackward() {
	s := lines.currentSection
	LIO.Debug("Reading backwards in section %v", s)

	if s.read.start == 0 {
		LIO.Trace("Section at the beginning of the buffer %v", s)
		return
	}

	// s.read.start points at the first character in the last line found string.
	// Decrementing pos by 1 will get us to point at the \n. Decrementing once again will
	// get us to look at the last character in the line we're reading.
	pos := s.read.start - 1
	if lines.data.At(pos) == '\n' {
		pos--
	}

	batchSize := lines.batchSize

	for pos > 0 && batchSize > 0 {
		if lines.data.At(pos) == '\n' || pos == 0 {
			line := Line{
				Offset:  pos + 1,
				index:   s.nextIndexBackward,
				section: lines.currentSection,
			}

			lines.lines.Insert(&line)
			LIO.Trace("Inserted line %v", &line)

			s.lines++
			s.nextIndexBackward--
			s.read.start = pos + 1

			batchSize--
		}
		pos--
	}
	s.read.start = pos

	LIO.Trace("Section after reading backward %v", s)
}

func (lines *Lines) first() *Line {
	LIO.Trace("Asked to return first in the buffer")

	if lines.data.fileSize == 0 {
		return nil
	}

	lines.jumpTo(0, "")
	t := lines.lines.Get(&Line{Offset: 0})
	if t != nil {
		LIO.Trace("Returning %v", *t)
		return *t
	}

	lines.readForward()
	t = lines.lines.Get(&Line{Offset: 0})
	if t != nil {
		LIO.Trace("Returning %v", *t)
		return *t
	}

	LIO.Trace("Returning nil")
	return nil
}

func (lines *Lines) last() *Line {
	size := lines.data.Size()

	LIO.Trace("Asked to return last in the buffer. Size %d", size)

	if size == 0 {
		LIO.Trace("The buffer is empty. Returning nil")
		return nil
	}

	lines.jumpTo(size, fmt.Sprintf("@%d", size))
	t := lines.lines.Predecessor(&Line{Offset: math.MaxInt64})
	if t != nil {
		LIO.Trace("Predecessor of infinity is %v", *t)
	}

	if t != nil {
		rawL := lines.RawLength(*t)
		if (*t).Offset+rawL >= size-1 {
			LIO.Trace("Returning %v", *t)
			return *t
		}
	}

	lines.readBackward()
	t = lines.lines.Predecessor(&Line{Offset: math.MaxInt64})
	if t != nil {
		LIO.Trace("Returning %v", *t)
		return *t
	}

	LIO.Trace("Retruning nil")
	return nil
}

func (lines *Lines) next(line *Line) *Line {
	LIO.Trace("Asked to return the next of %v. Data size %d", line, lines.data.Size())

	lines.currentSection = line.section

	var next *Line = nil
	t := lines.lines.Successor(line)
	if t != nil {
		next = *t
	}

	if next != nil {
		if next.section != line.section {
			rawL := lines.RawLength(line)
			if line.Offset+rawL == next.Offset {
				LIO.Trace("Returning %v", next)
				return next
			}
		} else {
			LIO.Trace("Returning %v", next)
			return next
		}
	}

	t = lines.lines.Successor(line)
	if t == nil || line.Offset+lines.RawLength(line) < (*t).Offset {
		lines.readForward()
		t = lines.lines.Successor(line)
	}

	if t != nil {
		LIO.Trace("Returning %v", *t)
		return *t
	}

	LIO.Trace("Returning nil")
	return nil
}

func (lines *Lines) prev(line *Line) *Line {
	LIO.Trace("Asked to return the previous of %v. Data size %d", line, lines.data.Size())

	lines.currentSection = line.section

	if line.Offset == 0 {
		LIO.Trace("Returning nil")
		return nil
	}

	var prev *Line = nil
	t := lines.lines.Predecessor(line)
	if t != nil {
		prev = *t
	}

	if prev != nil {
		if prev.section != line.section {
			rawL := lines.RawLength(prev)
			if prev.Offset+rawL == line.Offset {
				LIO.Trace("Returning %v", prev)
				return prev
			}
		} else {
			LIO.Trace("Returning %v", prev)
			return prev
		}
	}

	t = lines.lines.Predecessor(line)
	if t == nil || (*t).Offset+lines.RawLength(*t) < line.Offset {
		lines.readBackward()
		t = lines.lines.Predecessor(line)
	}

	if t != nil {
		LIO.Trace("Returning %v", *t)
		return *t
	}

	LIO.Trace("Returning nil")
	return nil
}

func (lines *Lines) jumpTo(offset int64, name string) {
	LIO.Trace("Asked to jump to offset %d (under name %s)", offset, name)

	s := Section{read: Range{start: offset, end: offset}}
	t := lines.sections.Get(&s)
	if t != nil {
		lines.currentSection = *t
		LIO.Trace("Current section after jumping to %d (%s) is %v", offset, name, lines.currentSection)
		return
	}

	t = lines.sections.Predecessor(&s)
	if t != nil {
		pred := *t
		if offset >= pred.read.start && offset < pred.read.end {
			lines.currentSection = pred
			LIO.Trace("Current section after jumping to %d (%s) is %v", offset, name, lines.currentSection)
			return
		} else if offset == pred.read.end {
			lines.currentSection = pred
			LIO.Trace("Current section after jumping to %d (%s) is %v", offset, name, lines.currentSection)
			return
		}
	}

	section := newSection(offset, name)

	lines.currentSection = section
	lines.sections.Insert(section)

	LIO.Trace("Current section after jumping to %d (%s) is %v", offset, name, lines.currentSection)
}

func (lines *Lines) at(offset int64) *Line {
	LIO.Trace("Asked for a line at offset %d", offset)

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
