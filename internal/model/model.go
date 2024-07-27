package model

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/help"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/jumpparser"
	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SearchDirection int

const (
	SEARCH_DOWN    = SearchDirection(iota)
	SEARCH_UP      = SearchDirection(1)
	SEARCH_NOWHERE = SearchDirection(2)
)

type Position struct {
	Line    *lines.Line
	Segment int
}

type Search struct {
	regex   bool
	pattern string
}

type SearchResult struct {
	start int
	end   int
}

type SearchRange struct {
	start int64
	end   int64
}

type CegmentRange struct {
	start int
	end   int
	style *lipgloss.Style
}

type ProgressReport struct {
	progress    float64
	interrupted bool
}

type BufferRefreshSignal int

func (c CegmentRange) Start() int {
	return c.start
}

func (c CegmentRange) End() int {
	return c.end
}

func (c CegmentRange) Subrange(start int, end int) utils.Range {
	return CegmentRange{start: start, end: end, style: c.style}
}

// / Colored segment of a string
type Cegment struct {
	subsegments []string
	style       []*lipgloss.Style
}

type Model struct {
	ready bool

	buffer                  *lines.Buffer
	lines                   *lines.Lines
	backgroundReaderStarted bool

	marks *avltree.AVLTree[*lines.Line]

	top            Position
	prevCursorLine *lines.Line
	cursorLine     *lines.Line

	width  int
	height int

	defaultStyle lipgloss.Style
	cursorStyle  lipgloss.Style
	errorStyle   lipgloss.Style
	statusStyle  lipgloss.Style
	promptStyle  lipgloss.Style
	editorStyle  lipgloss.Style
	markStyle    lipgloss.Style

	searchStyles [9]lipgloss.Style

	activeSearchTextbox   bool
	activeSearchDirection SearchDirection
	textbox               textinput.Model
	newSearch             Search

	searches     [9]Search
	searchHits   [9]*avltree.AVLTree[*lines.Line]
	searchRanges [9]*avltree.AVLTree[SearchRange]
	search       int
	prevSearch   int

	activeGotoTextbox bool
	gotoLine          int64

	inProgress        bool
	currentProgress   float64
	progressInterrupt chan bool
	progressCmd       tea.Cmd
	progressPrompt    string

	activeError           bool
	errorText             string
	errorTimeoutScheduled bool

	History *History

	helpActive   bool
	helpViewport viewport.Model

	currentWord      int
	currentWordStyle lipgloss.Style

	viewCommand bool

	activeShowTextbox bool
	showPattern       string

	activeHideTextbox bool
	hidePattern       string
}

var ML *hhlog.Logger
var SL *hhlog.Logger
var VL *hhlog.Logger

var searchForwardRunes = map[string]int{
	"1": 0,
	"2": 1,
	"3": 2,
	"4": 3,
	"5": 4,
	"6": 5,
	"7": 6,
	"8": 7,
	"9": 8,
}

var searchBackwardRunes = map[string]int{
	"!": 0,
	"@": 1,
	"#": 2,
	"$": 3,
	"%": 4,
	"^": 5,
	"&": 6,
	"*": 7,
	"(": 8,
}

func int64Comp(a, b int64) int {
	if a < b {
		return -1
	} else if a == b {
		return 0
	} else {
		return 1
	}
}

func searchRangeComp(a, b SearchRange) int {
	return int64Comp(a.start, b.start)
}

func InitialModel(b *lines.Buffer) *Model {
	m := Model{
		ready: false,

		buffer: b,
		lines:  lines.NewLines(b),

		top:        Position{nil, 0},
		cursorLine: nil,

		marks: avltree.NewAVLTree(lines.LineComp),

		defaultStyle: lipgloss.NewStyle(),
		cursorStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#ffda45")).
			Foreground(lipgloss.Color("#000000")),
		errorStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#ff0000")).
			Foreground(lipgloss.Color("#ffffff")),
		statusStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#eeeeee")).
			Foreground(lipgloss.Color("#000000")),
		editorStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#ffffff")),
		promptStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#ffffff")),
		markStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#00a0e4")),

		searchStyles: [9]lipgloss.Style{
			lipgloss.NewStyle().Background(lipgloss.Color("#00ff1a")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#00ef98")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#00ddbd")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#00ccd0")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#00bbde")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#00a9ed")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#0091ff")).Foreground(lipgloss.Color("#ffffff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#4469f2")).Foreground(lipgloss.Color("#ffffff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#7464d8")).Foreground(lipgloss.Color("#ffffff")),
		},

		searches:     [9]Search{},
		searchHits:   [9]*avltree.AVLTree[*lines.Line]{},
		searchRanges: [9]*avltree.AVLTree[SearchRange]{},

		textbox: textinput.New(),

		History: NewHistory(),

		helpViewport: viewport.New(0, 0),

		currentWord: -1,
		currentWordStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#ff7777")).
			Foreground(lipgloss.Color("#ffffff")),
	}

	m.search = 0
	m.cursorLine = m.lines.First()
	m.top.Line = m.cursorLine

	for i := 0; i < len(m.searches); i++ {
		m.searches[i] = Search{
			regex:   false,
			pattern: "",
		}
		m.searchHits[i] = avltree.NewAVLTree(lines.LineComp)
		m.searchRanges[i] = avltree.NewAVLTree(searchRangeComp)
	}

	return &m
}

func (m *Model) hasNext(pos Position) bool {
	segs := m.segmentize(pos.Line)
	if pos.Segment < len(segs)-1 {
		return true
	}
	if m.lines.Next(pos.Line) != nil {
		return true
	}
	return false
}

func (m *Model) next(pos Position) Position {
	segs := m.segmentize(pos.Line)
	if pos.Segment < len(segs)-1 {
		pos.Segment++
		return pos
	}

	next := m.lines.Next(pos.Line)
	if next != nil {
		pos.Line = next
		pos.Segment = 0
		return pos
	}

	panic("Attempt to move past the end of the buffer")
}

func (m *Model) hasPrev(pos Position) bool {
	if m.lines.Prev(pos.Line) == nil && pos.Segment == 0 {
		return false
	}
	return true
}

func (m *Model) prev(pos Position) Position {
	if pos.Segment > 0 {
		pos.Segment--
		return pos
	}

	prev := m.lines.Prev(pos.Line)
	if prev != nil {
		pos.Line = prev
		segs := m.segmentize(pos.Line)
		pos.Segment = len(segs) - 1
		return pos
	}

	panic("Attempt to move before beginning of the buffer")
}

// / Return lowest visible line and lowest visible segment in the line.
func (m *Model) bottomLine() (*lines.Line, int) {
	pos := Position{m.top.Line, m.top.Segment}
	for h := 0; m.hasNext(pos) && h < m.height-1; h++ {
		pos = m.next(pos)
	}
	return pos.Line, pos.Segment
}

func (m *Model) scrollOneSegmentDown() {
	if m.hasNext(m.top) {
		m.top = m.next(m.top)
	}
}

func (m *Model) onKeyUp() {
	if m.cursorLine != m.top.Line {
		m.cursorLine = m.lines.Prev(m.cursorLine)
	} else {
		if m.top.Line == nil {
			return
		}

		if m.top.Segment > 0 {
			m.top.Segment -= 1
		} else {
			if m.top.Line.Offset > 0 {
				newTop := m.lines.Prev(m.top.Line)
				if newTop != nil {
					m.top.Line = newTop
					segs := m.segmentize(m.top.Line)
					m.top.Segment = len(segs) - 1
					m.cursorLine = m.top.Line
				}
			}
		}
	}
}

func (m *Model) onKeyDown() {
	if m.lines.Next(m.cursorLine) != nil {
		bottomLine, bottomSegment := m.bottomLine()
		ML.Info("Scrolling down. Top line is %v, cursor %v, bottom %v(%d)",
			m.top, m.cursorLine, bottomLine, bottomSegment)
		if m.cursorLine == bottomLine {
			m.scrollOneSegmentDown()
			bottomLine, bottomSegment = m.bottomLine()
			ML.Info("New bottom %v(%d)", bottomLine, bottomSegment)
			if m.cursorLine.Offset < bottomLine.Offset {
				m.cursorLine = m.lines.Next(m.cursorLine)
			}
		} else {
			m.cursorLine = m.lines.Next(m.cursorLine)
		}
	} else {
		bottomLine, bottomSegment := m.bottomLine()
		segs := m.segmentize(bottomLine)
		if bottomSegment < len(segs)-1 {
			m.scrollOneSegmentDown()
		}
	}
}

func (m *Model) onKeyLeft() {
	line := m.lines.Text(m.cursorLine)
	words := strings.Fields(line)
	m.currentWord -= 1
	if m.currentWord < 0 {
		m.currentWord = len(words) - 1
	}
}

func (m *Model) onKeyRight() {
	line := m.lines.Text(m.cursorLine)
	words := strings.Fields(line)
	m.currentWord += 1
	if m.currentWord >= len(words) {
		m.currentWord = 0
	}
}

func (m *Model) onAddCurrentWord() {
	if m.currentWord == -1 {
		return
	}

	line := m.lines.Text(m.cursorLine)
	words := strings.Fields(line)
	word := words[m.currentWord]

	m.searches[m.search].regex = true
	if m.searches[m.search].pattern == "" {
		m.searches[m.search].pattern = word
	} else {
		m.searches[m.search].pattern += "|" + word
	}
}

func (m *Model) hideBeforeCursor() {
	m.top.Line = m.cursorLine
	m.top.Segment = 0
	m.lines.SetFirst(m.top.Line)
}

func (m *Model) hideAfterCursor() {
	m.lines.SetLast(m.cursorLine)
}

func (m *Model) hideReset() {
	m.lines.SetFirst(nil)
	m.lines.SetLast(nil)
}

func (m *Model) startProgress(prompt string, progressCmd tea.Cmd) tea.Cmd {
	m.progressInterrupt = make(chan bool)
	m.inProgress = true
	m.progressPrompt = prompt
	m.progressCmd = progressCmd
	return progressCmd
}

func (m *Model) onPageUp() {
	it := Position{m.cursorLine, 0}
	for i := 0; m.hasPrev(it) && i < m.height-1; i++ {
		it = m.prev(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onPageDown() {
	it := Position{m.cursorLine, 0}
	for i := 0; m.hasNext(it) && i < m.height-1; i++ {
		it = m.next(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onHalfPageUp() {
	it := Position{m.cursorLine, 0}
	for i := 0; m.hasPrev(it) && i < m.height/2; i++ {
		it = m.prev(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onHalfPageDown() {
	it := Position{m.cursorLine, 0}
	for i := 0; m.hasNext(it) && i < m.height/2; i++ {
		it = m.next(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onToggleMark() {
	ML.Info("Toggling mark at position %v", m.cursorLine)
	if m.marks.Contains(m.cursorLine) {
		m.marks.Remove(m.cursorLine)
	} else {
		m.marks.Insert(m.cursorLine)
	}
}

func (m *Model) searchMatches(line string, pat string, regex bool) []SearchResult {
	res := make([]SearchResult, 0)

	if pat != "" {
		if regex {
			re := regexp.MustCompile(pat)
			matches := re.FindAllStringIndex(line, -1)
			for _, match := range matches {
				res = append(res, SearchResult{start: match[0], end: match[1]})
			}
		} else {
			start := 0
			for {
				index := strings.Index(line[start:], pat)
				if index == -1 {
					break
				}
				matchStart := start + index
				matchEnd := matchStart + len(pat)
				res = append(res, SearchResult{start: matchStart, end: matchEnd})
				start = matchEnd
			}
		}
	}

	return res
}

func (m *Model) updateSearchRange(search int, r SearchRange) {
	if r.end == r.start {
		return
	}

	SL.Trace("Updating search range %v", r)

	tree := m.searchRanges[search]
	t := tree.Get(r)
	if t != nil {
		if t.end > r.end {
			r.end = t.end
		}

		if r.start != t.start || r.end != t.end {
			tree.Remove(r)
			tree.Insert(r)
			SL.Trace("New range %v", r)
		}
	} else {
		pred := tree.Predecessor(r)
		if pred != nil {
			// Ranges intersect
			if pred.start <= r.end && r.start <= pred.end {
				if r.start < pred.start {
					r.start = pred.start
				}

				if pred.end > r.end {
					r.end = pred.end
				}
				tree.Remove(*pred)
				SL.Trace("Removed range %v", *pred)
			}
		}
		SL.Trace("Insert range %v", r)
		tree.Insert(r)
	}
}

func (m *Model) searchScreen(search int, from *lines.Line, howMany int) *lines.Line {
	curr := m.cursorLine
	r := SearchRange{from.Offset, 0}
	for curr != nil && howMany >= 0 {
		matches := m.searchMatches(m.lines.Text(curr), m.searches[search].pattern, m.searches[search].regex)
		if len(matches) > 0 {
			m.searchHits[search].Insert(curr)
		}
		curr = m.lines.Next(curr)
		howMany--
	}

	if curr == nil {
		r.end = m.buffer.Size()
	} else {
		r.end = curr.Offset
	}

	m.updateSearchRange(search, r)

	if curr != nil {
		return curr
	} else {
		return m.cursorLine
	}
}

func (m *Model) findNextWithProgress(search int) tea.Cmd {
	searchLine := m.lines.Next(m.cursorLine)
	lastProgress := float64(0)

	SL.Info("Searching forward from %v. Looking for %v", searchLine, m.searches[search])

	return func() tea.Msg {
		SL.Debug("Callback called for %v", searchLine)

		if searchLine == nil || m.searches[search].pattern == "" {
			return ProgressReport{
				progress:    1.0,
				interrupted: true,
			}
		}

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(100000)
		var prog float64 = 0
		interrupted := false
		r := SearchRange{searchLine.Offset, 0}

		found := false
	outer:
		for searchLine != nil && batch > 0 {
			select {
			case interrupted = <-m.progressInterrupt:
				SL.Debug("Search interrupted")
				break outer
			default:
				SL.Trace("Checking line %v for match", searchLine)
				matches := m.searchMatches(m.lines.Text(searchLine),
					m.searches[search].pattern, m.searches[search].regex)
				if len(matches) > 0 {
					m.searchHits[search].Insert(searchLine)
					found = true
					break outer
				}
				searchLine = m.lines.Next(searchLine)
				batch--
			}
		}

		if found {
			interrupted = true
			m.cursorLine = searchLine
		} else {
			if searchLine == nil {
				prog = 1.0
				interrupted = true
			} else {
				prog = float64(searchLine.Offset) / float64(m.buffer.Size())
			}
		}

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		if searchLine != nil {
			r.end = searchLine.Offset
		} else {
			r.end = m.buffer.Size()
		}
		m.updateSearchRange(search, r)

		res := ProgressReport{
			progress:    prog,
			interrupted: interrupted,
		}

		SL.Debug("Returning progress report %v", res)

		return res
	}
}

func (m *Model) findPrevWithProgress(search int) tea.Cmd {
	searchLine := m.lines.Prev(m.cursorLine)
	lastProgress := float64(0)

	SL.Info("Searching backwards from %v. Searching for %v", searchLine, m.searches[search])

	return func() tea.Msg {
		SL.Debug("Callback called for %v", searchLine)

		if searchLine == nil || m.searches[search].pattern == "" {
			return ProgressReport{
				progress:    0,
				interrupted: true,
			}
		}

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(100000)
		var prog float64 = 0
		interrupted := false
		r := SearchRange{searchLine.Offset, 0}

		found := false
	outer:
		for searchLine != nil && batch > 0 {
			select {
			case interrupted = <-m.progressInterrupt:
				SL.Debug("Search interrupted")
				break outer
			default:
				SL.Trace("Checking line %v for match", searchLine)

				matches := m.searchMatches(m.lines.Text(searchLine),
					m.searches[search].pattern, m.searches[search].regex)
				if len(matches) > 0 {
					m.searchHits[search].Insert(searchLine)
					found = true
					SL.Debug("Line %v matches", searchLine)
					break outer
				}
				searchLine = m.lines.Prev(searchLine)
				batch--
			}
		}

		if found {
			interrupted = true
			m.cursorLine = searchLine
		} else {
			if searchLine == nil {
				prog = 1.0
				interrupted = true
			} else {
				prog = 1 - (float64(searchLine.Offset) / float64(m.buffer.Size()))
			}
		}

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		if searchLine == nil {
			r.end = m.buffer.Size()
		} else {
			r.end = searchLine.Offset
		}

		m.updateSearchRange(search, r)

		res := ProgressReport{
			progress:    prog,
			interrupted: interrupted,
		}

		SL.Debug("Returning progress report %v", res)

		return res
	}
}

func (m *Model) setError(err string) {
	m.activeError = true
	m.errorText = err
	m.errorTimeoutScheduled = false
}

func (m *Model) gotoWithProgress() tea.Cmd {
	SL.Info("Going to line %v", m.gotoLine)

	counter := int64(1)
	currentLine := m.lines.First()

	return func() tea.Msg {
		SL.Debug("Callback called for %v", counter)

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(100000)
		var prog float64 = 0
		interrupted := false

	outer:
		for batch > 0 && counter < m.gotoLine && currentLine != nil {
			select {
			case interrupted = <-m.progressInterrupt:
				SL.Debug("Goto interrupted")
				break outer
			default:
				currentLine = m.lines.Next(currentLine)
				batch--
				counter++
			}
		}

		if counter == m.gotoLine {
			interrupted = true
			m.cursorLine = currentLine
			m.centerCursor()
		} else if currentLine == nil {
			m.setError("Line not found")
			interrupted = true
		}

		prog = float64(counter) / float64(m.gotoLine)

		res := ProgressReport{
			progress:    prog,
			interrupted: interrupted,
		}

		SL.Debug("Returning progress report %v", res)

		return res
	}
}

// Given current position of the cursor, make sure it is visible.
// This is needed in cases when change of terminal's size causes the cursor
// to become invisible (usually by being below bottom line).
func (m *Model) normalizeView() {
	if !m.ready || m.cursorLine == nil {
		return
	}

	if m.top.Line.Offset > m.cursorLine.Offset {
		ML.Info("Cursor %v appears to be before top line %v", m.cursorLine, m.top.Line)
		m.top.Line = m.cursorLine
		m.top.Segment = 0
		ML.Info("Fixed top line to %v", m.top)
		return
	}

	bottomLine, _ := m.bottomLine()
	if m.cursorLine.Offset > bottomLine.Offset {
		ML.Info("Cursor %v appears to be after bottom line %v", m.cursorLine, bottomLine)
		pos := Position{m.cursorLine, len(m.segmentize(m.cursorLine)) - 1}
		for h := m.height; m.hasPrev(pos) && h > 1; h-- {
			pos = m.prev(pos)
		}
		m.top = pos
		ML.Info("Fixed top line to %v", m.top)
	}
}

func (m *Model) centerCursor() {
	segs := m.segmentize(m.cursorLine)

	pos := Position{
		Line:    m.cursorLine,
		Segment: 0,
	}
	for i := m.height/2 - len(segs)/2; m.hasPrev(pos) && i > 0; i-- {
		pos = m.prev(pos)
	}

	m.top = pos
	m.normalizeView()
}

func (m *Model) fixCursor() {
	pred := m.cursorLine
	succ := m.lines.Next(m.cursorLine)
	for pred != nil && succ != nil {
		if m.lines.Visible(pred) {
			m.cursorLine = pred
			break
		}

		if m.lines.Visible(succ) {
			m.cursorLine = succ
			break
		}

		pred = m.lines.Prev(pred)
		succ = m.lines.Next(succ)
	}

	m.centerCursor()
}

func (m *Model) nextMark() {
	succ := m.marks.Successor(m.cursorLine)
	if succ != nil {
		m.cursorLine = *succ
		m.centerCursor()
	}
}

func (m *Model) prevMark() {
	pred := m.marks.Predecessor(m.cursorLine)
	if pred != nil {
		m.cursorLine = *pred
		m.centerCursor()
	}
}

func (m *Model) segmentize(l *lines.Line) []string {
	line := m.lines.Text(l)
	res := make([]string, 0)
	if line == "" {
		seg := fmt.Sprintf("%-*s", m.width, " ")
		res = append(res, seg)
	} else {
		for line != "" {
			l := utils.Min(m.width, len(line))
			seg := line[:l]
			if len(seg) < m.width {
				seg = fmt.Sprintf("%-*s", m.width, seg)
			}
			res = append(res, seg)
			line = line[l:]
		}
	}
	return res
}

func (m *Model) cegmentize(l *lines.Line, baseStyle *lipgloss.Style, boldWord int, boldStyle *lipgloss.Style) []Cegment {
	segs := m.segmentize(l)
	r0 := make([]utils.Range, 0)

	line := m.lines.Text(l)
	start := 0
	for _, seg := range segs {
		r0 = append(r0, CegmentRange{start: start, end: start + len(seg), style: baseStyle})
		start += len(seg)
	}

	var boldWordStart, boldWordEnd int
	var err error
	if boldWord != -1 {
		boldWordStart, boldWordEnd, err = utils.NthWord(line, boldWord)
		if err == nil {
			r2 := make([]utils.Range, 0)
			r2 = append(r2, CegmentRange{start: boldWordStart, end: boldWordEnd, style: boldStyle})
			r0 = utils.RangesProduct(r0, r2)
		}
	}

	for i := 0; i < len(m.searches); i++ {
		if len(m.searches[i].pattern) == 0 {
			continue
		}

		matches := m.searchMatches(line, m.searches[i].pattern, m.searches[i].regex)
		if len(matches) == 0 {
			continue
		}

		r1 := make([]utils.Range, 0)
		for _, match := range matches {
			r1 = append(r1, CegmentRange{start: match.start, end: match.end, style: &m.searchStyles[i]})
		}

		r0 = utils.RangesProduct(r0, r1)

		if boldWord >= 0 {
			r2 := make([]utils.Range, 0)
			r2 = append(r2, CegmentRange{start: boldWordStart, end: boldWordEnd, style: boldStyle})
			r0 = utils.RangesProduct(r0, r2)
		}
	}

	res := make([]Cegment, len(segs))
	res[0].style = make([]*lipgloss.Style, 0)
	res[0].subsegments = make([]string, 0)
	seg := 0
	start = 0
	for _, rr := range r0 {
		r := rr.(CegmentRange)

		res[seg].style = append(res[seg].style, r.style)
		l := r.end - r.start
		res[seg].subsegments = append(res[seg].subsegments, segs[seg][start:start+l])

		start += l
		if start >= len(segs[seg]) {
			start -= len(segs[seg])
			seg++
		}
	}

	return res
}

func (m *Model) renderStatusBar() string {
	VL.Debug("Rendering status bar with cursor %v", m.cursorLine)

	if m.cursorLine == nil {
		st1 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, "EMPTY BUFFER"))
		st2 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))
		return st1 + "\n" + st2
	} else {
		VL.Trace("Cursor is not nil")
	}

	cont := m.marks.Contains(m.cursorLine)
	pred := m.marks.Predecessor(m.cursorLine)
	succ := m.marks.Successor(m.cursorLine)

	ll := ""
	ll = fmt.Sprintf("Line %s @%d ", m.cursorLine.Name(), m.cursorLine.Offset)
	tt := utils.ExtractTimestamp(m.lines.Text(m.cursorLine))
	if tt != nil {
		ll += tt.String()
	}

	ms := make([]string, 0)
	if pred != nil {
		ms = append(ms, fmt.Sprintf("↑%s", lines.Distance(*pred, m.cursorLine)))
		ttt := utils.ExtractTimestamp(m.lines.Text(*pred))
		if tt != nil && ttt != nil {
			ms = append(ms, (*tt).Sub(*ttt).String())
		}
	}
	if cont {
		ms = append(ms, "†")
	}
	if succ != nil {
		ms = append(ms, fmt.Sprintf("%s↓", lines.Distance(*succ, m.cursorLine)))
		ttt := utils.ExtractTimestamp(m.lines.Text(*succ))
		if tt != nil && ttt != nil {
			ms = append(ms, (*ttt).Sub(*tt).String())
		}
	}
	marks := strings.Join(ms, " ")

	rest := fmt.Sprintf("%-*s",
		m.width-(utf8.RuneCountInString(ll)+utf8.RuneCountInString(marks)), "")

	if marks != "" {
		ll += " "
	}

	if m.viewCommand {
		ll += " VIEW "
	}

	st1 := m.statusStyle.Render(ll)
	st1 += m.markStyle.Render(marks)
	st1 += m.statusStyle.Render(rest)

	st2 := ""

	if m.activeError {
		st2 += m.errorStyle.Render(fmt.Sprintf("%-*s", m.width, m.errorText))
	} else {
		st2 = m.statusStyle.Render("")

		for i := 0; i < len(m.searches); i++ {
			if m.searches[i].pattern == "" {
				continue
			}

			// var pred *lines.Line = nil
			pred = m.searchHits[i].Predecessor(m.cursorLine)
			if pred != nil && !m.lines.Visible(*pred) {
				pred = nil
			}

			succ = m.searchHits[i].Successor(m.cursorLine)
			if succ != nil && !m.lines.Visible(*succ) {
				succ = nil
			}

			sh := make([]string, 0)
			if pred != nil {
				sh = append(sh, fmt.Sprintf("↑%s", lines.Distance(*pred, m.cursorLine)))
				ttt := utils.ExtractTimestamp(m.lines.Text(*pred))
				if ttt != nil && tt != nil {
					sh = append(sh, (tt.Sub(*ttt).String()))
				}
			}

			if succ != nil {
				sh = append(sh, fmt.Sprintf("%s↓", lines.Distance(*succ, m.cursorLine)))
				ttt := utils.ExtractTimestamp(m.lines.Text(*succ))
				if ttt != nil && tt != nil {
					sh = append(sh, (ttt.Sub(*tt).String()))
				}
			}

			sh = append(sh, fmt.Sprintf("◎%d", m.searchHits[i].Count()))

			text := strings.Join(sh, " ")

			if i == m.search {
				m.searchStyles[i].Bold(true)
				m.searchStyles[i].Underline(true)
			}

			st2 += m.searchStyles[i].Render(text)
			m.searchStyles[i].Bold(false)
			m.searchStyles[i].Underline(false)

			st2 += m.statusStyle.Render(" ")
		}

		st2 += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))
	}

	return st1 + "\n" + st2
}

func (m *Model) renderProgress() string {
	res := ""

	text := "Press Ctrl+c or Esc to interrupt"
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	text = fmt.Sprintf("%s: %.1f%% done", m.progressPrompt, m.currentProgress*100)
	res += "\n"
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	return res
}

func (m *Model) renderShowBar() string {
	res := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, " "))

	m.textbox.Width = m.width
	m.textbox.Prompt = prompt
	m.textbox.PromptStyle = m.promptStyle
	m.textbox.TextStyle = m.editorStyle
	res += "\n" + m.textbox.View()

	return res
}

func (m *Model) renderHideBar() string {
	res := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, " "))

	prompt := "Hide (regex): "
	m.textbox.Width = m.width
	m.textbox.Prompt = prompt
	m.textbox.PromptStyle = m.promptStyle
	m.textbox.TextStyle = m.editorStyle
	res += "\n" + m.textbox.View()

	return res
}

func (m *Model) renderSearchBar() string {
	res := ""

	text := "Tab: (regex/plain)  F2/F4: Next/Prev"
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	prompt := fmt.Sprintf("%d: ", m.search+1)

	if m.newSearch.regex {
		prompt += "(regex): "
	} else {
		prompt += "(plain): "
	}

	m.textbox.Width = m.width
	m.textbox.Prompt = prompt
	m.textbox.PromptStyle = m.searchStyles[m.search]
	m.textbox.TextStyle = m.searchStyles[m.search]

	res += "\n" + m.textbox.View()

	return res
}

func (m *Model) renderGotoBar() string {
	res := ""

	text := ""
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	m.textbox.Width = m.width
	m.textbox.Prompt = ": "
	m.textbox.PromptStyle = m.promptStyle
	m.textbox.TextStyle = m.editorStyle

	res += "\n" + m.textbox.View()

	return res
}

func (m *Model) jumpTo(j jumpparser.JumpLocation) tea.Cmd {
	if j.Offset != nil {
		if *j.Offset < 0 || *j.Offset > 100 {
			m.setError("Invalid offset")
			return nil
		}

		if *j.Offset == 100 {
			m.cursorLine = m.lines.Last()
			m.centerCursor()
		} else if *j.Offset == 0 {
			m.cursorLine = m.lines.First()
			m.centerCursor()
		} else {
			where := int64(float64(m.buffer.Size()) * *j.Offset / 100)
			first := m.lines.First()
			last := m.lines.Last()
			if where >= first.Offset && where < last.Offset+last.RawLength {
				m.cursorLine = m.lines.At(where)
				m.centerCursor()
			} else {
				m.setError("The offset is outside of visible portion of the file")
			}
		}
	}

	if j.Lines != nil {
		m.gotoLine = int64(*j.Lines)
		return m.startProgress("Counting", m.gotoWithProgress())
	}

	return nil
}

func (m *Model) Init() tea.Cmd {
	m.History.Load()
	return m.helpViewport.Init()
}

func (m *Model) BufferRefresher() tea.Msg {
	for {
		if m.buffer.Refresh() {
			return BufferRefreshSignal(1)
		}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	viewCommand := m.viewCommand
	m.viewCommand = false

	ML.Info("Got request %v %v", reflect.TypeOf(msg), msg)

	cmds := make([]tea.Cmd, 0)

	if !m.backgroundReaderStarted {
		m.backgroundReaderStarted = true
		cmds = append(cmds, m.BufferRefresher)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if m.helpActive && k == "q" {
			m.helpActive = false
			break
		}

		if m.helpActive {
			if k == "q" {
				m.helpActive = false
				break
			}

			m.helpViewport, cmd = m.helpViewport.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			if m.activeShowTextbox {
				if k == "esc" || k == "ctrl+c" {
					m.activeShowTextbox = false
					m.textbox.SetValue("")
					m.textbox.Blur()
					m.History.ResetCursor()
				} else if k == "enter" {
					m.activeShowTextbox = false
					m.textbox.Blur()
					m.showPattern = m.textbox.Value()
					m.History.Append(m.textbox.Value())
					m.lines.SetShowPattern(m.showPattern)
					m.fixCursor()
				} else if k == "up" {
					m.textbox.SetValue(m.History.Prev(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else if k == "down" {
					m.textbox.SetValue(m.History.Next(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else {
					m.textbox, cmd = m.textbox.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else if m.activeHideTextbox {
				if k == "esc" || k == "ctrl+c" {
					m.activeHideTextbox = false
					m.textbox.SetValue("")
					m.textbox.Blur()
					m.History.ResetCursor()
				} else if k == "enter" {
					m.activeHideTextbox = false
					m.textbox.Blur()
					m.hidePattern = m.textbox.Value()
					m.History.Append(m.textbox.Value())
					m.lines.SetHidePattern(m.hidePattern)
					m.fixCursor()
				} else if k == "up" {
					m.textbox.SetValue(m.History.Prev(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else if k == "down" {
					m.textbox.SetValue(m.History.Next(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else {
					m.textbox, cmd = m.textbox.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else if m.activeGotoTextbox {
				if k == "esc" || k == "ctrl+c" {
					m.textbox.SetValue("")
					m.activeGotoTextbox = false
					m.textbox.Blur()
					m.History.ResetCursor()
				} else if k == "enter" {
					m.activeGotoTextbox = false
					m.textbox.Blur()

					m.History.Append(m.textbox.Value())

					j, err := jumpparser.Parse(m.textbox.Value())
					if err != nil {
						m.setError(fmt.Sprintf("%s", err))
					} else {
						cmd = m.jumpTo(j)
						cmds = append(cmds, cmd)
					}
				} else if k == "up" {
					m.textbox.SetValue(m.History.Prev(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else if k == "down" {
					m.textbox.SetValue(m.History.Next(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else {
					m.textbox, cmd = m.textbox.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else if m.activeSearchTextbox {
				if k == "enter" {
					m.activeSearchTextbox = false
					m.textbox.Blur()
					m.newSearch.pattern = m.textbox.Value()
					m.History.Append(m.textbox.Value())
					if m.newSearch.pattern != m.searches[m.search].pattern ||
						m.newSearch.regex != m.searches[m.search].regex {
						m.searches[m.search] = m.newSearch
						m.searchHits[m.search] = avltree.NewAVLTree(lines.LineComp)
						m.searchRanges[m.search] = avltree.NewAVLTree(searchRangeComp)
					}

					if m.activeSearchDirection == SEARCH_DOWN {
						cmd = m.startProgress("Searching", m.findNextWithProgress(m.search))
						cmds = append(cmds, cmd)
					} else if m.activeSearchDirection == SEARCH_UP {
						cmd = m.startProgress("Searching", m.findPrevWithProgress(m.search))
						cmds = append(cmds, cmd)
					}
				} else if k == "tab" {
					m.newSearch.regex = !m.newSearch.regex
				} else if k == "esc" || k == "ctrl+c" {
					m.search = m.prevSearch
					m.textbox.SetValue(m.searches[m.search].pattern)
					m.activeSearchTextbox = false
					m.textbox.Blur()
					m.History.ResetCursor()
				} else if k == "f4" {
					if m.newSearch.pattern != m.searches[m.search].pattern ||
						m.newSearch.regex != m.searches[m.search].regex {
						m.searches[m.search] = m.newSearch
						m.searchHits[m.search] = avltree.NewAVLTree(lines.LineComp)
						m.searchRanges[m.search] = avltree.NewAVLTree(searchRangeComp)
					}
					if m.search < len(m.searches)-1 {
						m.search++
						m.newSearch = m.searches[m.search]
						m.textbox.SetValue(m.newSearch.pattern)
					}
				} else if k == "f2" {
					if m.newSearch.pattern != m.searches[m.search].pattern ||
						m.newSearch.regex != m.searches[m.search].regex {
						m.searches[m.search] = m.newSearch
						m.searchHits[m.search] = avltree.NewAVLTree(lines.LineComp)
						m.searchRanges[m.search] = avltree.NewAVLTree(searchRangeComp)
						m.textbox.SetCursor(len(m.newSearch.pattern))
					}
					if m.search > 0 {
						m.search--
						m.newSearch = m.searches[m.search]
						m.textbox.SetValue(m.newSearch.pattern)
						m.textbox.SetCursor(len(m.newSearch.pattern))
					}
				} else if k == "up" {
					m.textbox.SetValue(m.History.Prev(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else if k == "down" {
					m.textbox.SetValue(m.History.Next(m.textbox.Value()))
					m.textbox.CursorEnd()
				} else {
					m.textbox, cmd = m.textbox.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else {
				if k == "q" || k == "ctrl+c" {
					if k == "ctrl+c" && m.inProgress {
						ML.Debug("Interrupting progress")
						m.progressInterrupt <- true
					} else {
						m.History.Save()
						return m, tea.Quit
					}
				}

				if k == "esc" && m.inProgress {
					m.progressInterrupt <- true
				}

				if k == "up" || k == "k" {
					m.onKeyUp()
				}

				if k == "down" || k == "j" {
					m.onKeyDown()
				}

				if k == "g" {
					m.cursorLine = m.lines.First()
				}

				if k == "G" {
					m.cursorLine = m.lines.Last()
				}

				if k == "R" {
					m.buffer.Refresh()
				}

				if k == "pgup" {
					m.onPageUp()
				}

				if k == "pgdown" || k == " " {
					m.onPageDown()
				}

				if k == "f" {
					m.onHalfPageDown()
				}

				if k == "enter" {
					m.centerCursor()
				}

				if k == "m" {
					m.onToggleMark()
				}

				if k == "." {
					m.nextMark()
				}

				if k == "," {
					m.prevMark()
				}

				if k == "e" {
					m.prevSearch = m.search

					m.newSearch = m.searches[m.search]
					m.textbox.SetValue(m.newSearch.pattern)
					m.textbox.SetCursor(len(m.newSearch.pattern))

					m.activeSearchTextbox = true
					cmds = append(cmds, m.textbox.Focus())
					cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
					m.activeSearchDirection = SEARCH_NOWHERE
				}

				if k == "f2" || k == "[" {
					m.onPrevSearch()
				}

				if k == "f4" || k == "]" {
					m.onNextSearch()
				}

				if k == "/" || k == "?" {
					if m.searches[m.search].pattern != "" && m.searchHits[m.search].Count() != 0 {
						m.prevSearch = m.search
						m.search = (m.search + 1) % len(m.searches)
					}

					m.newSearch = m.searches[m.search]
					m.textbox.SetValue(m.newSearch.pattern)
					m.textbox.SetCursor(len(m.newSearch.pattern))

					m.activeSearchTextbox = true
					cmds = append(cmds, m.textbox.Focus())
					cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
					if k == "/" {
						m.activeSearchDirection = SEARCH_DOWN
					}

					if k == "?" {
						m.activeSearchDirection = SEARCH_UP
					}
				}

				if k == ":" {
					m.activeGotoTextbox = true

					m.textbox.SetValue("")
					m.textbox.Prompt = ""

					cmds = append(cmds, m.textbox.Focus())
					cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
				}

				if k == "n" {
					cmd = m.startProgress("Searching", m.findNextWithProgress(m.search))
					cmds = append(cmds, cmd)
				}

				if k == "N" {
					cmd = m.startProgress("Searching", m.findPrevWithProgress(m.search))
					cmds = append(cmds, cmd)
				}

				if k == "h" && !viewCommand {
					ML.Info("Setting help viewport's text to: %s", help.Text)
					m.helpViewport.SetContent(help.Text)
					m.helpActive = true
				}

				if search, ok := searchForwardRunes[k]; ok {
					if m.searches[search].pattern != "" {
						cmd = m.startProgress("Searching", m.findNextWithProgress(search))
						cmds = append(cmds, cmd)
					}
				}

				if search, ok := searchBackwardRunes[k]; ok {
					if m.searches[search].pattern != "" {
						cmd = m.startProgress("Searching", m.findPrevWithProgress(search))
						cmds = append(cmds, cmd)
					}
				}

				if k == "left" {
					m.onKeyLeft()
				}

				if k == "right" {
					m.onKeyRight()
				}

				if k == "a" {
					if viewCommand {
						m.hideAfterCursor()
					} else {
						if m.searches[m.search].pattern != "" && m.searchHits[m.search].Count() != 0 {
							m.search = (m.search + 1) % len(m.searches)
						}
						m.onAddCurrentWord()
					}
				}

				if k == "b" {
					if viewCommand {
						m.hideBeforeCursor()
					} else {
						m.onHalfPageUp()
					}
				}

				if k == "c" && viewCommand {
					m.hideReset()
					m.lines.SetShowPattern("")
					m.lines.SetHidePattern("")
				}

				if k == "A" {
					m.onAddCurrentWord()
				}

				if k == "s" && viewCommand {
					m.activeShowTextbox = true
					m.textbox.Prompt = "regex"
					m.textbox.SetValue(m.showPattern)
					m.textbox.SetCursor(len(m.showPattern))
					cmds = append(cmds, m.textbox.Focus())
					cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
				}

				if k == "h" && viewCommand {
					m.activeHideTextbox = true
					m.textbox.Prompt = "regex"
					m.textbox.SetValue(m.hidePattern)
					m.textbox.SetCursor(len(m.hidePattern))
					cmds = append(cmds, m.textbox.Focus())
					cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
				}

				if k == "v" {
					m.viewCommand = true
				}
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.width = msg.Width
			m.height = msg.Height - 2
			m.ready = true
		} else {
			m.width = msg.Width
			m.height = msg.Height - 2
		}

		m.top.Segment = 0

		m.helpViewport, cmd = m.helpViewport.Update(msg)
		cmds = append(cmds, cmd)

		m.helpViewport.Width = msg.Width
		m.helpViewport.Height = msg.Height

	case ProgressReport:
		if !msg.interrupted {
			m.currentProgress = msg.progress
			cmds = append(cmds, m.progressCmd)
		} else {
			m.inProgress = false
			m.progressInterrupt = nil
		}

	case BufferRefreshSignal:
		ML.Debug("Buffer changed. New buffer is %d bytes long", m.buffer.Size())

		if m.ready && m.top.Line == nil {
			m.top.Line = m.lines.First()
			m.top.Segment = 0
			m.cursorLine = m.top.Line
		}

		cmds = append(cmds, m.BufferRefresher)
	}

	if m.activeError && !m.errorTimeoutScheduled {
		ML.Debug("Scheduling error timeout")
		m.errorTimeoutScheduled = true
		cmds = append(cmds, tea.Tick(time.Second, func(time.Time) tea.Msg {
			ML.Debug("Error timeout called")
			m.activeError = false
			return true
		}))
	}

	m.normalizeView()

	if m.prevCursorLine != m.cursorLine {
		m.prevCursorLine = m.cursorLine
		m.currentWord = -1
	}

	ML.Debug("Returning commands %v", cmds)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.helpActive {
		return m.helpViewport.View()
	}

	h := m.height
	w := m.width
	res := ""
	renderedLines := 0

	current := m.top
	desc := ""

	VL.Debug("Rendering the view from line %v, cursor %v, dimensions %dx%d",
		current, m.cursorLine, w, h)

	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern != "" {
			m.searchScreen(i, current.Line, h)
		}
	}

	for h > 0 {
		if current.Line == nil {
			break
		}

		var cegs []Cegment
		if m.cursorLine == current.Line {
			cegs = m.cegmentize(current.Line, &m.cursorStyle, m.currentWord, &m.currentWordStyle)
		} else {
			if m.marks.Contains(current.Line) {
				cegs = m.cegmentize(current.Line, &m.markStyle, -1, &m.currentWordStyle)
			} else {
				cegs = m.cegmentize(current.Line, &m.defaultStyle, -1, &m.currentWordStyle)
			}
		}

		for _, ceg := range cegs[current.Segment:] {
			if h == 0 {
				break
			}

			seg := ""

			for i := 0; i < len(ceg.subsegments); i++ {
				seg += ceg.style[i].Render(ceg.subsegments[i])
			}

			if res == "" {
				res = seg
			} else {
				res = res + "\n" + seg
			}

			renderedLines++
			desc = desc + fmt.Sprintf("%d ", current.Line.Offset)
			h -= 1
		}

		current.Line = m.lines.Next(current.Line)
		current.Segment = 0
	}

	VL.Trace("Rendered %d lines so far", renderedLines)

	for renderedLines < m.height {
		res += "\n"
		renderedLines++
	}

	res += "\n"

	if m.activeShowTextbox {
		res += m.renderShowBar()
	} else if m.activeHideTextbox {
		res += m.renderHideBar()
	} else if m.activeSearchTextbox {
		res += m.renderSearchBar()
	} else if m.activeGotoTextbox {
		res += m.renderGotoBar()
	} else if m.inProgress {
		res += m.renderProgress()
	} else {
		res += m.renderStatusBar()
	}

	VL.Debug("Rendered: %s, %d lines of text", strings.Trim(desc, " "), renderedLines)
	VL.Debug("Total rendered %d characters and %d lines", len(res), utils.CountLines(res))

	return res
}
