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
	lines := &Lines{
		data:           data,
		sections:       avltree.NewAVLTree[*Section](SectionComp),
		lines:          avltree.NewAVLTree[*Line](LineComp),
		currentSection: newSection(0, ""),
		view:           NewView(),
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
	LL.Debug("Reading forwards in section %v from offset %d", s, s.read.end)

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

			break
		}
		pos++
	}
	s.read.end = pos + 1
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

	if lines.view.forcedStart != 0 {
		return *lines.view.lines.Successor(&Line{Offset: 0})
	}

	if lines.view.visibleStart != 0 {
		size := lines.data.Size()
		lines.jumpTo(size, fmt.Sprintf("@%d", size))

		res := lines.view.lines.Successor(&Line{Offset: 0})
		utils.Assert(res != nil)
		utils.Assert((*res).Offset == lines.view.visibleStart, "%d %v", lines.view.visibleStart, *res)
		return *res
	}

	res := lines.first(0)
	for res != nil && !lines.visible(res) {
		res = lines.next(res, 0)
	}

	if res != nil {
		lines.view.lines.Insert(res)
		lines.view.UpdateVisibleStart(res.Offset)
	}

	return res
}

func (lines *Lines) SetFirst(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	lines.view.Reset()
	lines.view.SetFirst(line)

	LL.Debug("Set first line to %v", line)
}

func (lines *Lines) SetLast(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	lines.view.Reset()
	lines.view.SetLast(line)

	LL.Debug("Set last line to %v", line)
}

func (lines *Lines) last(iteration int) *Line {
	utils.Assert(iteration < 100)

	size := lines.data.Size()
	lines.jumpTo(size, fmt.Sprintf("@%d", size))
	t := lines.lines.Predecessor(&Line{Offset: math.MaxInt64})
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

	if lines.view.forcedEnd != math.MaxInt64 {
		return *lines.view.lines.Predecessor(&Line{Offset: math.MaxInt64})
	}

	if lines.view.visibleEnd != math.MaxInt64 {
		size := lines.data.Size()
		lines.jumpTo(size, fmt.Sprintf("@%d", size))

		res := lines.view.lines.Predecessor(&Line{Offset: math.MaxInt64})
		utils.Assert(res != nil)
		utils.Assert((*res).Offset+(*res).RawLength == lines.view.visibleEnd,
			"%d %v", lines.view.visibleEnd, *res)
		return *res
	}

	res := lines.last(0)
	for res != nil && !lines.visible(res) {
		res = lines.prev(res, 0)
	}

	if res != nil {
		lines.view.lines.Insert(res)
		lines.view.UpdateVisibleEnd(res.Offset + res.RawLength)
	}

	return res
}

func (lines *Lines) next(line *Line, iterations int) *Line {
	// This is a mere protection against bugs. We should be able to iterate as many times as we
	// need.
	if iterations > 100 {
		return nil
	}

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

func (lines *Lines) Next(line *Line) (next *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return nil
	}

	end := line.Offset + line.RawLength
	if end >= lines.view.forcedEnd || end >= lines.view.visibleEnd {
		return nil
	}

	visibleLine := false
	if lines.visible(line) {
		lines.view.lines.Insert(line)
		visibleLine = true
	}

	nextPtr := lines.view.lines.Successor(line)
	if nextPtr == nil || (*nextPtr).section != line.section {
		next = lines.next(line, 0)
		for next != nil {
			if lines.visible(next) {
				break
			} else {
				next = lines.next(next, 0)
			}
		}

		if next != nil {
			lines.view.lines.Insert(next)
		} else if visibleLine {
			utils.Assert(next == nil)
			LL.Trace("Setting visible end to line %v. Prev end %d",
				line, lines.view.visibleEnd)
			lines.view.UpdateVisibleEnd(line.Offset + line.RawLength)
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

	if line == nil {
		return nil
	}

	if line.Offset <= lines.view.forcedStart || line.Offset <= lines.view.visibleStart {
		return nil
	}

	visibleLine := false
	if lines.visible(line) {
		lines.view.lines.Insert(line)
		visibleLine = true
	}

	prevPtr := lines.view.lines.Predecessor(line)
	if prevPtr == nil || (*prevPtr).section != line.section {
		prev = lines.prev(line, 0)
		for prev != nil {
			if lines.visible(prev) {
				break
			} else {
				prev = lines.prev(prev, 0)
			}
		}

		if prev != nil {
			lines.view.lines.Insert(prev)
		} else if visibleLine {
			utils.Assert(prev == nil)
			LL.Trace("Setting visible start to line %v. Prev start %d",
				line, lines.view.visibleStart)
			lines.view.UpdateVisibleStart(line.Offset)
		}
	} else {
		prev = *prevPtr
	}

	return prev
}

func (lines *Lines) jumpTo(offset int64, name string) {
	if lines.view.OutOfRange(offset) {
		return
	}

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

func (lines *Lines) JumpTo(offset int64, name string) {
	mx.Lock()
	defer mx.Unlock()

	lines.jumpTo(offset, name)
}

func (lines *Lines) at(offset int64) *Line {
	if lines.view.OutOfRange(offset) {
		return nil
	}

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

func (lines *Lines) At(offset int64) *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.OutOfRange(offset) {
		return nil
	}

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

	if line != nil {
		lines.view.lines.Insert(line)
	}

	return line
}

func (lines *Lines) visible(line *Line) bool {
	if lines.view.OutOfRange(line.Offset) {
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

	lines.view.Reset()
	LL.Debug("Set show pattern to %s", pat)
}

func (lines *Lines) SetHidePattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.view.hidePattern = regexp.MustCompile(pat)
	} else {
		lines.view.hidePattern = nil
	}

	lines.view.Reset()
	LL.Debug("Set hide pattern to %s", pat)
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
