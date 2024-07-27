package lines

import (
	"fmt"
	"math"
	"regexp"
	"sync"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
)

var LL *hhlog.Logger
var mx sync.Mutex

func SectionComp(a, b *Section) int {
	if a.read.start < b.read.start {
		return -1
	} else if a.read.start == b.read.start {
		return 0
	} else {
		return 1
	}
}

// Class representing a range of lines, relative to Buffer.
type Range struct {
	start int64
	end   int64
}

type Section struct {
	name string

	read  Range // Range in the Buffer we scanned looking for lines
	lines int64 // how many lines we found

	nextIndexForward  int64
	nextIndexBackward int64
}

type Line struct {
	Offset    int64
	Length    int64 // length of text
	RawLength int64 // raw length, with end of line
	index     int64 // index in section
	section   *Section
}

type View struct {
	start int64
	end   int64

	lines *avltree.AVLTree[*Line]

	showPattern *regexp.Regexp
	hidePattern *regexp.Regexp
}

type Lines struct {
	data *Buffer

	lines          *avltree.AVLTree[*Line]
	sections       *avltree.AVLTree[*Section]
	currentSection *Section

	view *View
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

func newSection(offset int64, name string) *Section {
	return &Section{
		name:              name,
		nextIndexForward:  1,
		nextIndexBackward: 0,
		read:              Range{start: offset, end: offset},
	}
}

func NewLines(data *Buffer) *Lines {
	view := &View{
		lines:       avltree.NewAVLTree[*Line](LineComp),
		start:       0,
		end:         math.MaxInt64,
		showPattern: nil,
		hidePattern: nil,
	}

	lines := &Lines{
		data:           data,
		sections:       avltree.NewAVLTree[*Section](SectionComp),
		lines:          avltree.NewAVLTree[*Line](LineComp),
		currentSection: newSection(0, ""),
		view:           view,
	}

	lines.sections.Insert(lines.currentSection)

	LL.Debug("Started new section %v", lines.currentSection)

	return lines
}

func (line Line) String() string {
	return fmt.Sprintf("[Line: @%d, len=%d, rlen=%d, index=%d, sec=%p]",
		line.Offset, line.Length, line.RawLength, line.index, line.section)
}

func (s Section) String() string {
	return fmt.Sprintf("[Section: name=%s, lines=%d, read=%d,%d index=%d,%d]",
		s.name, s.lines, s.read.start, s.read.end, s.nextIndexForward, s.nextIndexBackward)
}

func (lines *Lines) readForward() {
	s := lines.currentSection

	LL.Debug("Reading forward in section %v from offset %d", s, s.read.end)

	pos := s.read.end
	for pos < lines.data.Size() {
		if lines.data.At(pos) == '\n' {
			len := pos - s.read.end
			if pos > 0 && lines.data.At(pos-1) == '\r' {
				len--
			}

			line := Line{
				Offset:    s.read.end,
				Length:    len,
				index:     s.nextIndexForward,
				RawLength: pos - s.read.end + 1,
				section:   lines.currentSection,
			}

			lines.lines.Insert(&line)

			s.lines++
			s.nextIndexForward++

			LL.Trace("Added line %v", line)
			break
		}
		pos++
	}
	s.read.end = pos + 1

	LL.Debug("Section after reading %v", s)
}

func (lines *Lines) readBackward() {
	s := lines.currentSection
	if s.read.start == 0 {
		return
	}

	LL.Debug("Reading backward in section %v from offset %d", s, s.read.start)

	// s.read.start points at the first character in the last line found string.
	// Decrementing pos by 1 will get us to point at the \n. Decrementing once again will
	// get us to look at the last character in the line we're reading.
	pos := s.read.start - 2
	for pos >= 0 {
		if lines.data.At(pos) == '\n' {
			len := s.read.start - pos - 2
			if pos > 0 && lines.data.At(pos-1) == '\r' {
				len--
			}

			line := Line{
				Offset:    pos + 1,
				Length:    len,
				index:     s.nextIndexBackward,
				RawLength: s.read.start - pos - 1,
				section:   lines.currentSection,
			}

			lines.lines.Insert(&line)

			s.lines++
			s.nextIndexBackward--

			LL.Trace("Added line %v", line)
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

		LL.Trace("Added line %v", line)
	}

	s.read.start = pos + 1

	LL.Debug("Section after reading %v", s)
}

func (lines *Lines) first(iteration int) *Line {
	utils.Assert(iteration < 100)

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

func (lines *Lines) First() *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.start != 0 {
		return *lines.view.lines.Successor(&Line{Offset: 0})
	}

	res := lines.first(0)
	for res != nil && !lines.visible(res) {
		res = lines.next(res, 0)
	}
	lines.view.lines.Insert(res)

	LL.Debug("Asked to return the first line. Returning %v", res)
	return res
}

func (lines *Lines) SetFirst(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	LL.Debug("Setting the first line to %v", line)

	lines.view.lines = avltree.NewAVLTree[*Line](LineComp)

	lines.view.lines.Insert(line)
	lines.view.start = line.Offset
}

func (lines *Lines) SetLast(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	LL.Debug("Setting the last line to %v", line)

	lines.view.lines = avltree.NewAVLTree[*Line](LineComp)

	lines.view.lines.Insert(line)
	lines.view.end = line.Offset + line.RawLength
}

func (lines *Lines) last(iteration int) *Line {
	utils.Assert(iteration < 100)

	size := lines.data.Size()
	LL.Trace("Jumping to offset %d", size)
	lines.jumpTo(size, fmt.Sprintf("@%d", size))
	t := lines.lines.Predecessor(&Line{Offset: math.MaxInt64})
	if t != nil {
		LL.Trace("Found last line in current section %v", *t)
	}
	if t != nil && (*t).Offset+(*t).RawLength >= size {
		return *t
	} else {
		lines.readBackward()
		return lines.last(iteration + 1)
	}
}

func (lines *Lines) Last() *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.end != math.MaxInt64 {
		return *lines.view.lines.Predecessor(&Line{Offset: math.MaxInt64})
	}

	LL.Trace("Asked to return last line.")
	res := lines.last(0)
	for res != nil && !lines.visible(res) {
		res = lines.prev(res, 0)
	}
	lines.view.lines.Insert(res)

	LL.Debug("Asked to return last line. Returning %v", res)
	return res
}

func (lines *Lines) moveToSection(line *Line, section *Section) {
	utils.Assert(line.section != section)

	// Works only if the line is at the beginning of its section or at the end of its section.
	// s2 := line.section
	// utils.Assert(
	// 	// Line at the beginning ot its section
	// 	s2.parsed.start == line.Offset ||
	// 		// Line at the end of its section
	// 		s2.parsed.end == line.Offset+line.rawLength,
	// 	section, line, s2)

	// if int64(s2.parsed.start)+s2.offset == line.Offset {
	// 	// The line is the first in its section. We have to trim the section's buffer and
	// 	// (parsedStart, parsedEnd)
	// }
}

func (lines *Lines) next(line *Line, iterations int) *Line {
	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	if iterations > 100 {
		return nil
	}

	if line.Offset+int64(line.RawLength) >= lines.data.Size() {
		LL.Trace("Line is past end of file %d >= %d",
			line.Offset+int64(line.RawLength), lines.data.Size())
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
			if line.section != next.section {
				lines.moveToSection(next, line.section)
			}

			return next
		} else {
			lines.readForward()
			return lines.next(line, iterations+1)
		}
	}
}

func (lines *Lines) Next(line *Line) (next *Line) {
	mx.Lock()
	defer mx.Unlock()

	LL.Trace("Asked for next of %v", line)

	if line == nil {
		return nil
	}

	if line.Offset+line.RawLength >= lines.view.end {
		return nil
	}

	nextPtr := lines.view.lines.Successor(line)
	if nextPtr == nil || (*nextPtr).section != line.section {
		next = lines.next(line, 0)
		for next != nil {
			if lines.visible(next) {
				LL.Trace("Next of %v is visible %v", line, next)
				break
			} else {
				LL.Trace("Immediate next of %v is invisible %v. Checking next", line, next)
				next = lines.next(next, 0)
			}
		}

		if next != nil {
			lines.view.lines.Insert(next)
		}
	} else {
		next = *nextPtr
	}

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
		return lines.prev(line, iterations+1)
	} else {
		if prev.Offset+int64(prev.RawLength) == int64(line.Offset) {
			if line.section != prev.section {
				lines.moveToSection(line, prev.section)
			}
			return prev
		} else {
			lines.readBackward()
			return lines.prev(line, iterations+1)
		}
	}
}

func (lines *Lines) Prev(line *Line) (prev *Line) {
	mx.Lock()
	defer mx.Unlock()

	LL.Trace("Asked for prev of %v", line)

	if line == nil {
		return nil
	}

	if line.Offset < lines.view.start {
		return nil
	}

	prevPtr := lines.view.lines.Predecessor(line)
	if prevPtr == nil || (*prevPtr).section != line.section {
		prev = lines.prev(line, 0)
		for prev != nil {
			if lines.visible(prev) {
				LL.Trace("Prev of %v is visible %v", line, prev)
				break
			} else {
				LL.Trace("Immediate prev of %v is invisible %v. Checking prev", line, prev)
				prev = lines.prev(prev, 0)
			}
		}

		if prev != nil {
			lines.view.lines.Insert(prev)
		}
	} else {
		prev = *prevPtr
	}

	return prev
}

func (lines *Lines) jumpTo(offset int64, name string) {
	LL.Trace("Asked to jump to offset %d, name %s", offset, name)

	s := Section{read: Range{start: offset, end: offset}}
	t := lines.sections.Get(&s)
	if t != nil {
		lines.currentSection = *t
		LL.Trace("Set current section to %v when asked to jump to %s",
			lines.currentSection, name)
		return
	}

	t = lines.sections.Predecessor(&s)
	if t != nil {
		pred := *t
		if offset >= pred.read.start && offset < pred.read.end {
			lines.currentSection = pred
			LL.Trace("Set current section to %v when asked to jump to %s",
				lines.currentSection, name)
			return
		} else if offset == pred.read.end {
			lines.currentSection = pred
			LL.Trace("Set current section to %v when asked to jump to %s",
				lines.currentSection, name)
			return
		}
	}

	section := newSection(offset, name)

	lines.currentSection = section
	lines.sections.Insert(section)

	LL.Debug("Started new section %v", section)
}

func (lines *Lines) JumpTo(offset int64, name string) {
	mx.Lock()
	defer mx.Unlock()

	if offset < lines.view.start || offset >= lines.view.end {
		return
	}

	lines.jumpTo(offset, name)
}

func (lines *Lines) at(offset int64) *Line {
	if offset < lines.view.start || offset >= lines.view.end {
		return nil
	}

	name := fmt.Sprintf("@%d", offset)
	lines.jumpTo(offset, name)

	s := lines.currentSection

	t := lines.lines.Predecessor(&Line{Offset: offset})
	if t != nil && (*t).section == s {
		LL.Debug("Asked for line at offset of %d, with section name %s. Returning %+v",
			offset, name, *t)
		return *t
	}

	LL.Debug("Reading around in section %v from offset %d", s, s.read.end)

	// Read back until we run into end of line. Then read forward...
	pos := offset
	for pos >= 0 {
		if lines.data.At(pos) == '\n' {
			pos++
			break
		}
		pos--
	}

	LL.Trace("Found new line starting from position %d", pos)

	// Making sure that next line we add in readForward will get index 0. Forward index
	// is set to 1 by default to accomodate the first section. This does not work for
	// sections we add in the middle of the file.
	s.nextIndexForward = 0
	s.nextIndexBackward = -1

	s.read = Range{pos, pos}
	lines.readForward()

	t = lines.lines.Get(&Line{Offset: pos})
	utils.Assert(t != nil, offset, name)

	LL.Debug("Asked for line at offset of %d, with section name %s. Returning %+v",
		offset, name, *t)

	return *t
}

func (lines *Lines) At(offset int64) *Line {
	mx.Lock()
	defer mx.Unlock()

	LL.Debug("Asked to return the line at offset %d", offset)

	line := lines.at(offset)
	prev := line
	next := line
	for line != nil {
		if lines.visible(prev) {
			return prev
		} else {
			prev = lines.prev(prev, 0)
		}

		if lines.visible(next) {
			return next
		} else {
			next = lines.next(next, 0)
		}
	}

	lines.view.lines.Insert(line)

	LL.Debug("Returning %v", line)
	return line
}

func (lines *Lines) visible(line *Line) bool {
	if line.Offset < lines.view.start || line.Offset >= lines.view.end {
		return false
	}

	show := true
	if lines.view.showPattern != nil && !lines.view.showPattern.MatchString(lines.text(line)) {
		show = false
	}

	if lines.view.hidePattern != nil && lines.view.hidePattern.MatchString(lines.text(line)) {
		show = false
	}

	return show
}

func (lines *Lines) Visible(line *Line) bool {
	mx.Lock()
	defer mx.Unlock()

	return lines.visible(line)
}

func (lines *Lines) text(line *Line) string {
	return string(lines.data.Range(line.Offset, line.Offset+line.Length))
}

func (lines *Lines) Text(line *Line) string {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return ""
	}

	return lines.text(line)
}

func (lines *Lines) SetShowPattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.view.showPattern = regexp.MustCompile(pat)
	} else {
		lines.view.showPattern = nil
	}

	lines.view.lines = avltree.NewAVLTree[*Line](LineComp)
}

func (lines *Lines) SetHidePattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.view.hidePattern = regexp.MustCompile(pat)
	} else {
		lines.view.hidePattern = nil
	}

	lines.view.lines = avltree.NewAVLTree[*Line](LineComp)
}

func (line *Line) Name() string {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return ""
	}

	s := line.section
	if s.name == "" {
		return fmt.Sprintf("%d", line.index)
	}
	var t string
	if line.index == 0 {
		t = s.name
	} else {
		t = fmt.Sprintf("%s%+d", s.name, line.index)
	}
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
