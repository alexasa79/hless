package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alexasa79/hless/internal/avltree"
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
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#ffffff")),
		promptStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#eeeeee")).
			Foreground(lipgloss.Color("#000000")),
		markStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#00a0e4")),

		searchStyles: [9]lipgloss.Style{
			lipgloss.NewStyle().Background(lipgloss.Color("#f48fb1")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ce93d8")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#b39ddb")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#9fa8da")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#90caf9")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#81d4fa")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#80deea")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#80cbc4")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#a5d6a7")).Foreground(lipgloss.Color("#000000")),
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
	words := utils.SplitWords(line)
	m.currentWord -= 1
	if m.currentWord < 0 {
		m.currentWord = len(words) - 1
	}
}

func (m *Model) onKeyRight() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
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
	words := utils.SplitWords(line)
	if m.currentWord >= len(words) {
		m.currentWord = -1
		return
	}
	word := words[m.currentWord].Word

	m.searches[m.search].regex = true
	if m.searches[m.search].pattern == "" {
		m.searches[m.search].pattern = word
	} else {
		m.searches[m.search].pattern += "|" + word
	}
}

func (m *Model) onAddCurrentWordShowPattern() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	if m.currentWord >= len(words) {
		m.currentWord = -1
		return
	}
	if m.showPattern == "" {
		m.showPattern = words[m.currentWord].Word
	} else {
		m.showPattern = fmt.Sprintf("%s|%s", m.showPattern, words[m.currentWord].Word)
	}
	m.lines.SetShowPattern(m.showPattern)
	m.fixCursor()
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

	var boldCeg CegmentRange
	if boldWord != -1 {
		words := utils.SplitWords(line)
		if boldWord < len(words) {
			boldCeg = CegmentRange{start: words[boldWord].Start, end: words[boldWord].End, style: boldStyle}
			r2 := make([]utils.Range, 0)
			r2 = append(r2, boldCeg)
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
			r2 = append(r2, boldCeg)
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
