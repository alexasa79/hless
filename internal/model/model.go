package model

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type SearchHistory struct {
	Search []string
}

type History struct {
	Searches [9]SearchHistory
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

/// Colored segment of a string
type Cegment struct {
	subsegments []string
	style       []*lipgloss.Style
}

type Model struct {
	ready bool

	lines *lines.Lines

	marks *avltree.AVLTree[*lines.Line]

	top        Position
	cursorLine *lines.Line

	width  int
	height int

	defaultStyle lipgloss.Style
	cursorStyle  lipgloss.Style
	statusStyle  lipgloss.Style
	markStyle    lipgloss.Style

	searchStyles [9]lipgloss.Style

	activeSearchTextbox       bool
	activeSearchDirectionDown bool
	textbox                   textinput.Model
	newSearch                 Search

	searches     [9]Search
	searchHits   [9]*avltree.AVLTree[*lines.Line]
	searchRanges [9]*avltree.AVLTree[SearchRange]
	search       int

	inProgress        bool
	currentProgress   float64
	progressInterrupt chan bool
	progressCmd       tea.Cmd
	progressPrompt    string

	History History
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

func InitialModel(f io.ReadSeeker, size int64) *Model {
	m := Model{
		ready: false,

		lines: lines.NewLines(f, size),

		top:        Position{nil, 0},
		cursorLine: nil,

		marks: avltree.NewAVLTree(lines.LineComp),

		defaultStyle: lipgloss.NewStyle(),
		cursorStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#ffda45")).
			Foreground(lipgloss.Color("#000000")),
		statusStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#eeeeee")).
			Foreground(lipgloss.Color("#000000")),
		markStyle: lipgloss.
			NewStyle().
			Background(lipgloss.Color("#00a0e4")),

		searchStyles: [9]lipgloss.Style{
			lipgloss.NewStyle().Background(lipgloss.Color("#a50cfe")),
			lipgloss.NewStyle().Background(lipgloss.Color("#cd19ff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#f026ff")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ff32ee")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ff3fd2")),
			lipgloss.NewStyle().Background(lipgloss.Color("#fe4cba")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ff59a6")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ff6596")),
			lipgloss.NewStyle().Background(lipgloss.Color("#ff728a")),
		},

		searches:     [9]Search{},
		searchHits:   [9]*avltree.AVLTree[*lines.Line]{},
		searchRanges: [9]*avltree.AVLTree[SearchRange]{},

		textbox: textinput.New(),

		History: History{},
	}

	m.cursorLine = m.lines.First()
	m.top.Line = m.cursorLine

	for i := 0; i < len(m.searches); i++ {
		m.searches[i] = Search{
			regex:   false,
			pattern: "",
		}
		m.searchHits[i] = avltree.NewAVLTree(lines.LineComp)
		m.searchRanges[i] = avltree.NewAVLTree(searchRangeComp)

		m.History.Searches[i].Search = make([]string, 0)
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

/// Return lowest visible line and lowest visible segment in the line.
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
		if m.top.Segment > 0 {
			m.top.Segment -= 1
		} else {
			if m.top.Line.Offset > 0 {
				m.top.Line = m.lines.Prev(m.top.Line)
				segs := m.segmentize(m.top.Line)
				m.top.Segment = len(segs) - 1
				m.cursorLine = m.top.Line
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
		matches := m.searchMatches(curr.Text(), m.searches[search].pattern, m.searches[search].regex)
		if len(matches) > 0 {
			m.searchHits[search].Insert(curr)
		}
		curr = m.lines.Next(curr)
		howMany--
	}

	if curr == nil {
		r.end = m.lines.FileSize()
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
	if searchLine == nil {
		return nil
	}
	lastProgress := float64(0)

	SL.Info("Searching forward from %v", searchLine)

	return func() tea.Msg {
		SL.Debug("Callback called for %v", searchLine)

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(100000)
		var prog float64 = 0
		interrupted := false
		r := SearchRange{searchLine.Offset, 0}

		if !interrupted {
			select {
			case interrupted = <-m.progressInterrupt:
				break
			default:
				found := false
				for searchLine != nil && batch > 0 {
					matches := m.searchMatches(searchLine.Text(),
						m.searches[search].pattern, m.searches[search].regex)
					if len(matches) > 0 {
						m.searchHits[search].Insert(searchLine)
						found = true
						break
					}
					searchLine = m.lines.Next(searchLine)
					batch--
				}

				if found {
					interrupted = true
					m.cursorLine = searchLine
				} else {
					if searchLine == nil {
						prog = 1.0
						interrupted = true
					} else {
						prog = float64(searchLine.Offset) / float64(m.lines.FileSize())
					}
				}
			}
		}

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		if searchLine != nil {
			r.end = searchLine.Offset
		} else {
			r.end = m.lines.FileSize()
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
	if searchLine == nil {
		return nil
	}
	lastProgress := float64(0)

	SL.Info("Searching backwards from %v", searchLine)

	return func() tea.Msg {
		SL.Debug("Callback called for %v", searchLine)

		// One of two things can happen. Either we already read to the end of the file.
		// In this case we set the progress based on the current search line out of total
		// number of lines. Or we did not yet reach the end of the file. In this case
		// we set the progress to be the progress returned by populate()
		batch := int64(10000)
		var prog float64 = 0
		interrupted := false
		r := SearchRange{searchLine.Offset, 0}
		select {
		case interrupted = <-m.progressInterrupt:
			break
		default:
			found := false
			for searchLine != nil && batch > 0 {
				SL.Trace("Checking line %v for match", searchLine)

				matches := m.searchMatches(searchLine.Text(),
					m.searches[search].pattern, m.searches[search].regex)
				if len(matches) > 0 {
					m.searchHits[search].Insert(searchLine)
					found = true
					SL.Debug("Line %v matches", searchLine)
					break
				}
				searchLine = m.lines.Prev(searchLine)
				batch--
			}

			if found {
				interrupted = true
				m.cursorLine = searchLine
			} else {
				if searchLine == nil {
					prog = 1.0
					interrupted = true
				} else {
					prog = 1 - (float64(searchLine.Offset) / float64(m.lines.FileSize()))
				}
			}
		}

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		if searchLine == nil {
			r.end = m.lines.FileSize()
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

/// Given current position of the cursor, make sure it is visible.
/// This is needed in cases when change of terminal's size causes the cursor
/// to become invisible (usually by being below bottom line).
func (m *Model) normalizeView() {
	if !m.ready {
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
	line := l.Text()
	res := make([]string, 0)
	for line != "" {
		l := utils.Min(m.width, len(line))
		seg := line[:l]
		if len(seg) < m.width {
			seg = fmt.Sprintf("%-*s", m.width, seg)
		}
		res = append(res, seg)
		line = line[l:]
	}
	return res
}

func (m *Model) cegmentize(l *lines.Line, baseStyle *lipgloss.Style) []Cegment {
	segs := m.segmentize(l)
	r0 := make([]utils.Range, 0)

	line := l.Text()

	start := 0
	for _, seg := range segs {
		r0 = append(r0, CegmentRange{start: start, end: start + len(seg), style: baseStyle})
		start += len(seg)
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
	cont := m.marks.Contains(m.cursorLine)
	pred := m.marks.Predecessor(m.cursorLine)
	succ := m.marks.Successor(m.cursorLine)

	ll := ""
	// if m.lines.Eof() {
	// 	lines = fmt.Sprintf("Line %s/%s  Size %s bytes",
	// 		m.cursorLine.Name, utils.Sprint64(m.lines.Nlines()),
	// 		utils.Sprint64(m.lines.FileSize()))
	// } else {
	ll = fmt.Sprintf("Line %s/∞ ", m.cursorLine.Name())
	// }

	ms := make([]string, 0)
	if pred != nil {
		ms = append(ms, fmt.Sprintf("↑%s", lines.Distance(*pred, m.cursorLine)))
	}
	if cont {
		ms = append(ms, "†")
	}
	if succ != nil {
		ms = append(ms, fmt.Sprintf("%s↓", lines.Distance(*succ, m.cursorLine)))
	}
	marks := strings.Join(ms, " ")

	rest := fmt.Sprintf("%-*s",
		m.width-(utf8.RuneCountInString(ll)+utf8.RuneCountInString(marks)), "")

	st1 := m.statusStyle.Render(ll)
	st1 += m.markStyle.Render(marks)
	st1 += m.statusStyle.Render(rest)

	st2 := m.statusStyle.Render("")

	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern == "" {
			continue
		}

		pred = m.searchHits[i].Predecessor(m.cursorLine)
		succ = m.searchHits[i].Successor(m.cursorLine)

		sh := make([]string, 0)
		if pred != nil {
			r := m.searchRanges[i].Predecessor(SearchRange{(*pred).Offset, 0})
			if r != nil && (*pred).Offset >= r.start && (*pred).Offset < r.end {
				sh = append(sh, fmt.Sprintf("↑%s", lines.Distance(*pred, m.cursorLine)))
			} else {
				sh = append(sh, fmt.Sprintf("↑%s", lines.Distance(*pred, m.cursorLine)))
			}
		}

		if succ != nil {
			r := m.searchRanges[i].Predecessor(SearchRange{(*succ).Offset, 0})
			if r != nil && (*succ).Offset >= r.start && (*succ).Offset < r.end {
				sh = append(sh, fmt.Sprintf("%s↓", lines.Distance(*succ, m.cursorLine)))
			} else {
				sh = append(sh, fmt.Sprintf("%s↓", lines.Distance(*succ, m.cursorLine)))
			}
		}

		sh = append(sh, fmt.Sprintf("◎%d", m.searchHits[i].Count()))

		st2 += m.searchStyles[i].Render(strings.Join(sh, " "))
		st2 += m.statusStyle.Render(" ")
	}

	st2 += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))

	return "\n" + st1 + "\n" + st2
}

func (m *Model) renderProgress() string {
	res := ""

	res += "\n"

	text := "Press Ctrl+c or Esc to interrupt"
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	text = fmt.Sprintf("%s: %.1f%% done", m.progressPrompt, m.currentProgress*100)
	res += "\n"
	res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, text))

	return res
}

func (m *Model) renderSearchBar() string {
	res := ""

	text := "Tab: (regex/plain)  F2/F4: Next/Prev  Ctrl+n: Next empty"
	res += m.statusStyle.Render(fmt.Sprintf("\n%-*s", m.width, text))

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

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	ML.Info("Got request %v %v", reflect.TypeOf(msg), msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if m.activeSearchTextbox {
			if k == "enter" {
				m.activeSearchTextbox = false
				m.textbox.Blur()
				m.newSearch.pattern = m.textbox.Value()
				if m.newSearch.pattern != m.searches[m.search].pattern ||
					m.newSearch.regex != m.searches[m.search].regex {
					m.searches[m.search] = m.newSearch
					m.searchHits[m.search] = avltree.NewAVLTree(lines.LineComp)
					m.searchRanges[m.search] = avltree.NewAVLTree(searchRangeComp)
				}

				if m.activeSearchDirectionDown {
					cmd = m.startProgress("Searching", m.findNextWithProgress(m.search))
					cmds = append(cmds, cmd)
				} else {
					cmd = m.startProgress("Searching", m.findPrevWithProgress(m.search))
					cmds = append(cmds, cmd)
				}
			} else if k == "tab" {
				m.newSearch.regex = !m.newSearch.regex
			} else if k == "esc" || k == "ctrl+c" {
				m.textbox.SetValue(m.searches[m.search].pattern)
				m.activeSearchTextbox = false
				m.textbox.Blur()
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
			} else if k == "ctrl+n" {
				next := (m.search + 1) % len(m.searches)
				for i := 0; i < len(m.searches); i++ {
					if m.searches[next].pattern == "" {
						break
					}
					next = (next + 1) % len(m.searches)
				}
				if next != m.search {
					m.search = next
					m.newSearch = m.searches[m.search]
					m.textbox.SetValue(m.newSearch.pattern)
				}
			} else {
				m.textbox, cmd = m.textbox.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else {
			if k == "q" || k == "ctrl+c" {
				if k == "ctrl+c" && m.inProgress {
					m.progressInterrupt <- true
				} else {
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

			if k == "pgup" {
				m.onPageUp()
			}

			if k == "pgdown" || k == " " {
				m.onPageDown()
			}

			if k == "f" {
				m.onHalfPageDown()
			}

			if k == "b" {
				m.onHalfPageUp()
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

			if k == "/" || k == "?" {
				m.activeSearchTextbox = true
				cmds = append(cmds, m.textbox.Focus())
				cmds = append(cmds, m.textbox.Cursor.BlinkCmd())
				if k == "/" {
					m.activeSearchDirectionDown = true
				}

				if k == "?" {
					m.activeSearchDirectionDown = false
				}
			}

			if k == "n" {
				cmd = m.startProgress("Searching", m.findNextWithProgress(m.search))
				cmds = append(cmds, cmd)
			}

			if k == "N" {
				cmd = m.startProgress("Searching", m.findPrevWithProgress(m.search))
				cmds = append(cmds, cmd)
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

	case ProgressReport:
		if !msg.interrupted {
			m.currentProgress = msg.progress
			cmds = append(cmds, m.progressCmd)
		} else {
			m.inProgress = false
			m.progressInterrupt = nil
		}
	}

	m.normalizeView()

	ML.Debug("Returning commands %v", cmds)

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	h := m.height
	w := m.width
	res := ""
	renderedLines := 0

	current := m.top
	desc := ""

	VL.Info("Rendering the view from line %v, cursor %v, dimensions %dx%d",
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
			cegs = m.cegmentize(current.Line, &m.cursorStyle)
		} else {
			if m.marks.Contains(current.Line) {
				cegs = m.cegmentize(current.Line, &m.markStyle)
			} else {
				cegs = m.cegmentize(current.Line, &m.defaultStyle)
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

	for renderedLines < m.height {
		res += fmt.Sprintf("%-*s\n", w, " ")
		renderedLines++
	}

	if m.activeSearchTextbox {
		res += m.renderSearchBar()
	} else if m.inProgress {
		res += m.renderProgress()
	} else {
		res += m.renderStatusBar()
	}

	VL.Info("Rendered: %s, %d lines", strings.Trim(desc, " "), renderedLines)

	return res
}
