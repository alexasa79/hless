package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/jumpparser"
	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

type MarkLine struct {
	line  *lines.Line
	style int
}

type Model struct {
	ready bool

	buffer *lines.Buffer
	lines  *lines.Lines

	marks *avltree.AVLTree[*MarkLine]

	top            Position
	prevCursorLine *lines.Line
	cursorLine     *lines.Line

	width  int
	height int

	defaultStyle     lipgloss.Style
	cursorStyle      lipgloss.Style
	errorStyle       lipgloss.Style
	statusStyle      lipgloss.Style
	promptStyle      lipgloss.Style
	editorStyle      lipgloss.Style
	markStyle        [3]lipgloss.Style
	textHeader1Style lipgloss.Style
	textHeader2Style lipgloss.Style
	boldTextStyle    lipgloss.Style

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
	gotoTime          time.Time
	gotoHow           jumpparser.RelativeJump

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

	selectionStart     int
	selectionEnd       int
	selectionFirstWord int
	currentWordStyle   lipgloss.Style

	viewCommand bool

	activeShowTextbox bool
	showPattern       string

	activeHideTextbox bool
	hidePattern       string

	inventoryActive   bool
	inventoryViewport viewport.Model
}

var ML *hhlog.Logger = hhlog.NewLogger("MODEL")
var SL *hhlog.Logger = hhlog.NewLogger("SEARCH")
var VL *hhlog.Logger = hhlog.NewLogger("VIEW")

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

func markLineComp(a, b *MarkLine) int {
	if a.line.Offset < b.line.Offset {
		return -1
	} else if a.line.Offset == b.line.Offset {
		return 0
	} else {
		return 1
	}
}

func InitialModel(b *lines.Buffer) *Model {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := Model{
		ready: false,

		buffer: b,
		lines:  lines.NewLines(b),

		top:        Position{nil, 0},
		cursorLine: nil,

		marks: avltree.NewAVLTree(markLineComp),

		defaultStyle: lipgloss.NewStyle(),
		cursorStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#ffda45")).
			Foreground(lipgloss.Color("#000000")),
		errorStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#ff0000")).
			Foreground(lipgloss.Color("#ffffff")),
		statusStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#eeeeee")).
			Foreground(lipgloss.Color("#000000")),
		editorStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#ffffff")),
		promptStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#eeeeee")).
			Foreground(lipgloss.Color("#000000")),
		markStyle: [3]lipgloss.Style{
			lipgloss.NewStyle().Background(lipgloss.Color("#1976d2")).Foreground(lipgloss.Color("#ffffff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#e65100")).Foreground(lipgloss.Color("#ffffff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#2e7d32")).Foreground(lipgloss.Color("#ffffff")),
		},
		textHeader1Style: lipgloss.NewStyle().
			Background(lipgloss.Color("#0d47a1")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true),
		textHeader2Style: lipgloss.NewStyle().
			Background(lipgloss.Color("#1e88e5")).
			Foreground(lipgloss.Color("#ffffff")),
		boldTextStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true),
		searchStyles: [9]lipgloss.Style{
			lipgloss.NewStyle().Background(lipgloss.Color("#ce93d8")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#b39ddb")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#9fa8da")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#bbdefb")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#81d4fa")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#80deea")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#80cbc4")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#a5d6a7")).Foreground(lipgloss.Color("#000000")),
			lipgloss.NewStyle().Background(lipgloss.Color("#dcedc8")).Foreground(lipgloss.Color("#000000")),
		},

		searches:     [9]Search{},
		searchHits:   [9]*avltree.AVLTree[*lines.Line]{},
		searchRanges: [9]*avltree.AVLTree[SearchRange]{},

		textbox: textinput.New(),

		History: NewHistory(),

		helpViewport: viewport.New(0, 0),

		selectionStart: -1,
		selectionEnd:   -1,

		currentWordStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#ff7777")).
			Foreground(lipgloss.Color("#ffffff")),

		inventoryViewport: viewport.New(0, 0),
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

	m.helpViewport.Style = m.helpViewport.Style.Border(lipgloss.ThickBorder())
	m.helpViewport.Style = m.helpViewport.Style.BorderForeground(lipgloss.Color("#ffffff"))

	m.inventoryViewport.Style = m.inventoryViewport.Style.Border(lipgloss.ThickBorder())
	m.inventoryViewport.Style = m.inventoryViewport.Style.BorderForeground(lipgloss.Color("#ffffff"))

	return &m
}

func (m *Model) hasNext(pos Position) bool {
	segs := m.segmentize(pos.Line, false)
	if pos.Segment < len(segs)-1 {
		return true
	}
	if m.lines.Next(pos.Line) != nil {
		return true
	}
	return false
}

func (m *Model) next(pos Position) Position {
	segs := m.segmentize(pos.Line, false)
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
		segs := m.segmentize(pos.Line, false)
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
					segs := m.segmentize(m.top.Line, false)
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
		segs := m.segmentize(bottomLine, false)
		if bottomSegment < len(segs)-1 {
			m.scrollOneSegmentDown()
		}
	}
}

func (m *Model) findSelectedWords(words []utils.Word) []int {
	findings := make([]int, 0)
	if m.selectionStart >= 0 {
		for i, w := range words {
			if w.Start <= m.selectionEnd && m.selectionStart <= w.End {
				findings = append(findings, i)
			}
		}
	}
	return findings
}

func (m *Model) onKeyLeft() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)
	if len(selectedWords) > 0 {
		firstWord := selectedWords[0]
		if firstWord > 0 {
			firstWord--
		} else {
			firstWord = len(words) - 1
		}
		m.selectionStart = words[firstWord].Start
		m.selectionEnd = words[firstWord].End
		m.selectionFirstWord = firstWord
	} else {
		if len(words) > 0 {
			m.selectionStart = words[len(words)-1].Start
			m.selectionEnd = words[len(words)-1].End
			m.selectionFirstWord = len(words) - 1
		}
	}
}

func (m *Model) onExpandSelectionLeft() {
	if m.selectionStart < 0 {
		return
	}

	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)
	nWords := len(selectedWords)
	if nWords == 0 || len(words) == 0 {
		return
	}

	firstSelectedWord := selectedWords[0]
	lastSelectedWord := selectedWords[len(selectedWords)-1]

	utils.Assert(m.selectionFirstWord == firstSelectedWord ||
		m.selectionFirstWord == lastSelectedWord)

	if nWords == 1 {
		m.selectionFirstWord = firstSelectedWord
		if m.selectionFirstWord > 0 {
			m.selectionStart = words[m.selectionFirstWord-1].Start
		}
	} else {
		if m.selectionFirstWord == firstSelectedWord {
			m.selectionEnd = words[lastSelectedWord-1].End
		} else {
			m.selectionStart = words[firstSelectedWord-1].Start
		}
	}
}

func (m *Model) onKeyRight() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)
	if len(selectedWords) > 0 {
		lastWord := selectedWords[len(selectedWords)-1]
		lastWord = (lastWord + 1) % len(words)
		m.selectionStart = words[lastWord].Start
		m.selectionEnd = words[lastWord].End
		m.selectionFirstWord = lastWord
	} else {
		if len(words) > 0 {
			m.selectionStart = words[0].Start
			m.selectionEnd = words[0].End
			m.selectionFirstWord = 0
		}
	}
}

func (m *Model) onExpandSelectionRight() {
	if m.selectionStart < 0 {
		return
	}

	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)
	nSelectedWords := len(selectedWords)
	if nSelectedWords == 0 || len(words) == 0 {
		return
	}

	firstSelectedWord := selectedWords[0]
	lastSelectedWord := selectedWords[len(selectedWords)-1]

	utils.Assert(m.selectionFirstWord == firstSelectedWord ||
		m.selectionFirstWord == lastSelectedWord)

	if nSelectedWords == 1 {
		m.selectionFirstWord = firstSelectedWord
		if m.selectionFirstWord < len(words)-1 {
			m.selectionEnd = words[m.selectionFirstWord+1].End
		}
	} else {
		if m.selectionFirstWord == firstSelectedWord {
			m.selectionEnd = words[lastSelectedWord+1].End
		} else {
			m.selectionStart = words[firstSelectedWord+1].Start
		}
	}
}

func (m *Model) onAddCurrentWord() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)

	if len(selectedWords) == 0 {
		return
	}

	start := words[selectedWords[0]].Start
	end := words[selectedWords[len(selectedWords)-1]].End

	m.searches[m.search].regex = true
	if m.searches[m.search].pattern == "" {
		m.searches[m.search].pattern = line[start:end]
	} else {
		m.searches[m.search].pattern += "|" + line[start:end]
	}

	m.selectionStart = -1
	m.selectionEnd = -1
}

func (m *Model) onAddCurrentWordShowPattern() {
	line := m.lines.Text(m.cursorLine)
	words := utils.SplitWords(line)
	selectedWords := m.findSelectedWords(words)

	if len(selectedWords) == 0 {
		return
	}

	start := words[selectedWords[0]].Start
	end := words[selectedWords[len(selectedWords)-1]].End

	if m.showPattern == "" {
		m.showPattern = line[start:end]
	} else {
		m.showPattern = fmt.Sprintf("%s|%s", m.showPattern, line[start:end])
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
	ml := m.marks.Get(&MarkLine{line: m.cursorLine})
	if ml != nil {
		m.marks.Remove(*ml)
		if (*ml).style < len(m.markStyle)-1 {
			n := &MarkLine{
				line:  m.cursorLine,
				style: ((*ml).style + 1) % len(m.markStyle),
			}
			m.marks.Insert(n)
		}
	} else {
		n := &MarkLine{
			line:  m.cursorLine,
			style: 0,
		}
		m.marks.Insert(n)
	}
}

func (m *Model) onClearMark() {
	ML.Info("Clearing mark at position %v", m.cursorLine)
	ml := m.marks.Get(&MarkLine{line: m.cursorLine})
	if ml != nil {
		m.marks.Remove(*ml)
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

	counter := int64(0)
	currentLine := m.lines.First()
	if m.gotoHow != jumpparser.JUMP_ABSOLUTE {
		currentLine = m.cursorLine
	}

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
				if m.gotoHow == jumpparser.JUMP_BACKWARD {
					currentLine = m.lines.Prev(currentLine)
				} else {
					currentLine = m.lines.Next(currentLine)
				}
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

func (m *Model) gotoTimeWithProgress() tea.Cmd {
	SL.Info("Going to time %v", m.gotoTime)

	currentLine := m.cursorLine

	return func() tea.Msg {
		cts := utils.ExtractTimestamp(m.lines.Text(currentLine))
		if cts != nil {
			SL.Debug("Current time is %v", *cts)
		}

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(10000)
		var prog float64 = 0
		interrupted := false
		finished := false

	outer:
		for batch > 0 && currentLine != nil {
			select {
			case interrupted = <-m.progressInterrupt:
				SL.Debug("Goto interrupted")
				break outer
			default:
				cts := utils.ExtractTimestamp(m.lines.Text(currentLine))
				if m.gotoHow == jumpparser.JUMP_BACKWARD {
					if cts != nil && m.gotoTime.Compare(*cts) >= 0 {
						finished = true
						break outer
					}
					currentLine = m.lines.Prev(currentLine)
				} else {
					if cts != nil && m.gotoTime.Compare(*cts) < 0 {
						finished = true
						break outer
					}
					currentLine = m.lines.Next(currentLine)
				}
				batch--
			}
		}

		if finished {
			interrupted = true
			m.cursorLine = currentLine
			m.centerCursor()
			prog = 1.0
		} else if currentLine == nil {
			m.setError("Line not found")
			interrupted = true
			prog = 1.0
		} else {
			if m.gotoHow == jumpparser.JUMP_BACKWARD {
				prog = 1.0 - (float64(currentLine.Offset) / float64(m.cursorLine.Offset))
			} else {
				prog = (float64(currentLine.Offset)-float64(m.cursorLine.Offset))/
					float64(m.buffer.Size()) - float64(m.cursorLine.Offset)
			}
		}

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
		pos := Position{m.cursorLine, len(m.segmentize(m.cursorLine, false)) - 1}
		for h := m.height; m.hasPrev(pos) && h > 1; h-- {
			pos = m.prev(pos)
		}
		m.top = pos
		ML.Info("Fixed top line to %v", m.top)
	}
}

func (m *Model) centerCursor() {
	segs := m.segmentize(m.cursorLine, false)

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
	ml := &MarkLine{line: m.cursorLine}
	succ := m.marks.Successor(ml)
	if succ != nil {
		m.cursorLine = (*succ).line
		m.centerCursor()
	}
}

func (m *Model) prevMark() {
	ml := &MarkLine{line: m.cursorLine}
	pred := m.marks.Predecessor(ml)
	if pred != nil {
		m.cursorLine = (*pred).line
		m.centerCursor()
	}
}

func (m *Model) segmentize(l *lines.Line, fill bool) []string {
	line := m.lines.Text(l)
	res := make([]string, 0)
	if line == "" {
		seg := fmt.Sprintf("%-*s", m.width, " ")
		res = append(res, seg)
	} else {
		for line != "" {
			l := utils.Min(m.width, len(line))
			seg := line[:l]
			if fill && len(seg) < m.width {
				seg = fmt.Sprintf("%-*s", m.width, seg)
			}
			res = append(res, seg)
			line = line[l:]
		}
	}
	return res
}

func (m *Model) cegmentize(l *lines.Line, baseStyle *lipgloss.Style, boldStart int, boldEnd int, boldStyle *lipgloss.Style, fill bool) []Cegment {
	segs := m.segmentize(l, fill)
	r0 := make([]utils.Range, 0)

	line := m.lines.Text(l)
	start := 0
	for _, seg := range segs {
		r0 = append(r0, CegmentRange{start: start, end: start + len(seg), style: baseStyle})
		start += len(seg)
	}

	var boldCeg CegmentRange
	if boldStart != -1 {
		boldCeg = CegmentRange{start: boldStart, end: boldEnd, style: boldStyle}
		r2 := make([]utils.Range, 0)
		r2 = append(r2, boldCeg)
		r0 = utils.RangesProduct(r0, r2)
	}

	for i := 0; i < len(m.searches); i++ {
		style := m.searchStyles[i].Copy()
		if i == m.search {
			style = style.Bold(true)
		}

		if len(m.searches[i].pattern) == 0 {
			continue
		}

		matches := m.searchMatches(line, m.searches[i].pattern, m.searches[i].regex)
		if len(matches) == 0 {
			continue
		}

		r1 := make([]utils.Range, 0)
		for _, match := range matches {
			r1 = append(r1, CegmentRange{start: match.start, end: match.end, style: &style})
		}

		r0 = utils.RangesProduct(r0, r1)

		if boldStart >= 0 {
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
			if where >= first.Offset && where < last.Offset+m.lines.RawLength(last) {
				m.cursorLine = m.lines.At(where)
				m.centerCursor()
			} else {
				m.setError("The offset is outside of visible portion of the file")
			}
		}
	}

	if j.Lines != nil {
		m.gotoLine = int64(*j.Lines)
		m.gotoHow = j.How
		return m.startProgress("Counting", m.gotoWithProgress())
	}

	if j.MicroSeconds != nil {
		cts := utils.ExtractTimestamp(m.lines.Text(m.cursorLine))
		if cts != nil {
			m.gotoTime = *cts
			m.gotoHow = j.How
			if m.gotoHow == jumpparser.JUMP_FORWARD {
				m.gotoTime = m.gotoTime.Add(time.Duration(*j.MicroSeconds))
			} else if m.gotoHow == jumpparser.JUMP_BACKWARD {
				m.gotoTime = m.gotoTime.Add(-time.Duration(*j.MicroSeconds))
			}
			return m.startProgress("Counting", m.gotoTimeWithProgress())
		} else {
			m.setError("Current line has no readable timestamp")
		}
	}

	return nil
}

func (m *Model) bufferRefresher() tea.Msg {
	sig, ok := <-m.buffer.BufferRefreshChannel
	if ok {
		return sig
	} else {
		return nil
	}
}

func (m *Model) Init() tea.Cmd {
	m.History.Load()
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, m.helpViewport.Init())
	cmds = append(cmds, m.inventoryViewport.Init())
	cmds = append(cmds, m.bufferRefresher)
	return tea.Batch(cmds...)
}
