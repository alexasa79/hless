package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Position struct {
	Line int
	Segment int
}

type Search struct {
	regex bool
	pattern string
	eof bool
}

type SearchResult struct {
	start int
	end int
}

type CegmentRange struct {
	start int
	end int
	style *lipgloss.Style
}

func (c CegmentRange) Start() int {
	return c.start
}

func (c CegmentRange) End() int {
	return c.end
}

func (c CegmentRange) Subrange(start int, end int) Range {
	return CegmentRange{ start: start, end: end, style: c.style }
}

/// Colored segment of a string
type Cegment struct {
	subsegments []string
	style []*lipgloss.Style
}

type Model struct {
	ready bool

	file io.Reader
	input chan string
	lines []string
	eof bool

	marks *AVLTree[int]

	top Position
	cursorLine int

	width int
	height int

	defaultStyle lipgloss.Style
	cursorStyle lipgloss.Style
	statusStyle lipgloss.Style
	markStyle lipgloss.Style

	searchStyles [9]lipgloss.Style

	activeSearchTextbox bool
	activeSearchDirectionDown bool
	textbox textinput.Model
	newSearch Search

	searches [9]Search
	searchHits [9]*AVLTree[int]
	search int
}

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

func intComp(a, b int) int {
	if a < b {
		return -1
	} else if a == b {
		return 0
	} else {
		return 1
	}
}

func initialModel(f io.Reader) *Model {
	m := Model {
		ready: false,

		file: f,
		input: make(chan string),
		lines: make([]string, 0),
		eof: false,

		top: Position{ 0, 0 },
		cursorLine: 0,

		marks: NewAVLTree[int](intComp),

		defaultStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(lipgloss.Color("#000")),
		cursorStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#ffda45")),
		statusStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#fff")),
		markStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#fff")).
			Background(lipgloss.Color("#00a0e4")),

		searchStyles: [9] lipgloss.Style {
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#7b00ff")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#d9ff00")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#2fff00")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#ff0084")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#4544c1")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#ff7b00")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#00fffb")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#84ff00")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#00ff7b")),
		},

		searches: [9] Search {},
		searchHits: [9] *AVLTree[int]{},

		textbox: textinput.New(),
	}

	for i := 0; i < len(m.searches); i++ {
		m.searches[i] = Search {
			regex: false,
			pattern: "",
		}
	}

	return &m
}

func (m *Model) hasNext(pos Position) bool {
	m.populate(pos.Line + 1)
	segs := m.segmentize(m.lines[pos.Line])
	if pos.Segment < len(segs) - 1 {
		return true
	}
	if pos.Line < len(m.lines) - 1 {
		return true
	}
	return false
}

func (m *Model) next(pos Position) Position {
	m.populate(pos.Line + 1)
	segs := m.segmentize(m.lines[pos.Line])
	if pos.Segment < len(segs) - 1 {
		pos.Segment++
		return pos
	}

	if pos.Line < len(m.lines) - 1 {
		pos.Line++
		pos.Segment = 0
		return pos
	}

	panic("Attempt to move past the end of the buffer")
}

func (m *Model) hasPrev(pos Position) bool {
	if pos.Line == 0 && pos.Segment == 0 {
		return false
	}
	return true
}

func (m *Model) prev(pos Position) Position {
	if pos.Segment > 0 {
		pos.Segment--
		return pos
	}

	if pos.Line > 0 {
		pos.Line--
		segs := m.segmentize(m.lines[pos.Line])
		pos.Segment = len(segs) - 1
		return pos
	}

	panic("Attempt to move before beginning of the buffer")
}

/// Return lowest visible line and lowest visible segment in the line.
func (m *Model) bottomLine() (int, int) {
	pos := Position { m.top.Line, m.top.Segment }
	for h := 0; m.hasNext(pos) && h < m.height - 1; h++ {
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
		m.cursorLine -= 1
	} else {
		if m.top.Segment > 0 {
			m.top.Segment -= 1
		} else {
			if m.top.Line > 0 {
				m.top.Line -= 1
				segs := m.segmentize(m.lines[m.top.Line])
				m.top.Segment = len(segs) - 1
				m.cursorLine = m.top.Line
			}
		}
	}
}

func (m *Model) onKeyDown() {
	h := m.height

	m.populate(m.cursorLine + h)

	if m.cursorLine < len(m.lines) - 1 {
		bottomLine, bottomSegment := m.bottomLine()
		log.Printf("Scrolling down. Top line is %v, cursor %d, bottom %d(%d)",
			m.top, m.cursorLine, bottomLine, bottomSegment)
		if m.cursorLine == bottomLine {
			m.scrollOneSegmentDown()
			bottomLine, bottomSegment = m.bottomLine()
			log.Printf("New bottom %d(%d)\n", bottomLine, bottomSegment)
			if m.cursorLine < bottomLine {
				m.cursorLine += 1
			}
		} else {
			m.cursorLine += 1
		}
	} else {
		bottomLine, bottomSegment := m.bottomLine()
		segs := m.segmentize(m.lines[bottomLine])
		if (bottomSegment < len(segs) - 1) {
			m.scrollOneSegmentDown()
		}
	}
}

func (m *Model) onDocumentEnd() {
	h := m.height

	m.populate(2147483647) // maximum value of int

	m.cursorLine = len(m.lines) - 1
	line := len(m.lines) - 1

	var segs []string
	for {
		segs = m.segmentize(m.lines[line])
		if h > len(segs) {
			h -= len(segs)
			line--
		} else {
			break
		}
	}

	m.top.Line = line
	m.top.Segment = len(segs) - h
}

func (m *Model) onPageUp() {
	m.populate(m.top.Line + m.height)

	it := Position{ m.cursorLine, 0 }
	for i := 0; m.hasPrev(it) && i < m.height - 1; i++ {
		it = m.prev(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onPageDown() {
	it := Position{ m.cursorLine, 0 }
	for i := 0; m.hasNext(it) && i < m.height - 1; i++ {
		it = m.next(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onHalfPageUp() {
	m.populate(m.top.Line + m.height)

	it := Position{ m.cursorLine, 0 }
	for i := 0; m.hasPrev(it) && i < m.height / 2; i++ {
		it = m.prev(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onHalfPageDown() {
	it := Position{ m.cursorLine, 0 }
	for i := 0; m.hasNext(it) && i < m.height / 2; i++ {
		it = m.next(it)
	}
	m.cursorLine = it.Line
}

func (m *Model) onToggleMark() {
	log.Printf("Toggling mark at position %d\n", m.cursorLine)
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

func (m *Model) findNext(search int, limit int) int {
	curr := m.cursorLine + 1
	m.populate(curr + 1)
	for curr < len(m.lines) && curr < limit {
		m.populate(curr + 2)

		matches := m.searchMatches(m.lines[curr], m.searches[search].pattern, m.searches[search].regex)
		if len(matches) > 0 {
			m.searchHits[search].Insert(curr)
			break
		}
		curr++
	}

	if curr == len(m.lines) {
		m.searches[search].eof = true
	}

	if curr < len(m.lines) {
		return curr
	} else {
		return m.cursorLine
	}
}

func (m *Model) findPrev(search int) int {
	curr := m.cursorLine - 1
	m.populate(curr - 1)
	for curr >= 0 {
		matches := m.searchMatches(m.lines[curr], m.searches[search].pattern, m.searches[search].regex)
		if len(matches) > 0 {
			m.searchHits[search].Insert(curr)
			break
		}
		curr--
	}

	if curr >= 0 {
		return curr
	} else {
		return m.cursorLine
	}
}

/// Given current position of the cursor, make sure it is visible.
/// This is needed in cases when change of terminal's size causes the cursor
/// to become invisible (usually by being below bottom line).
func (m *Model) normalizeView() {
	if !m.ready {
		return
	}

	m.populate(m.top.Line + m.height)

	if m.top.Line > m.cursorLine {
		log.Printf("Cursor %d appears to be before top line %d", m.cursorLine, m.top.Line)
		m.top.Line = m.cursorLine
		m.top.Segment = 0
		log.Printf("Fixed top line to %v", m.top)
		return
	}

	bottomLine, _ := m.bottomLine()
	if m.cursorLine > bottomLine {
		log.Printf("Cursor %d appears to be after bottom line %d", m.cursorLine, bottomLine)
		pos := Position { m.cursorLine, len(m.segmentize(m.lines[m.cursorLine])) - 1 }
		for h := m.height; m.hasPrev(pos) && h > 1; h-- {
			pos = m.prev(pos)
		}
		m.top = pos
		log.Printf("Fixed top line to %v", m.top)
	}
}

func (m *Model) centerCursor() {
	m.populate(m.cursorLine + m.height)

	segs := m.segmentize(m.lines[m.cursorLine])

	pos := Position {
		Line: m.cursorLine,
		Segment: 0,
	}
	for i := m.height / 2 - len(segs) / 2; m.hasPrev(pos) && i > 0; i-- {
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

func (m *Model) populate(n int) {
	if m.eof {
		return
	}

	for len(m.lines) < n {
		if line, ok := <- m.input; ok {
			m.lines = append(m.lines, line)
		} else {
			m.eof = true
			break
		}
	}
}

func (m *Model) segmentize(line string) []string {
	res := make([]string, 0)
	for line != "" {
		l := min(m.width, len(line))
		seg := line[:l]
		if len(seg) < m.width {
			seg = fmt.Sprintf("%-*s", m.width, seg)
		}
		res = append(res, seg)
		line = line[l:]
	}
	return res
}

func (m *Model) cegmentize(line string, baseStyle *lipgloss.Style) []Cegment {
	segs := m.segmentize(line)
	r0 := make([]Range, 0)

	start := 0
	for _, seg := range segs {
		r0 = append(r0, CegmentRange{ start: start, end: start + len(seg), style: baseStyle })
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

		r1 := make([]Range, 0)
		for _, match := range matches {
			r1 = append(r1, CegmentRange{ start: match.start, end: match.end, style: &m.searchStyles[i] })
		}

		r0 = RangesProduct(r0, r1)
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
		res[seg].subsegments = append(res[seg].subsegments, segs[seg][start:start + l])

		start += l
		if start >= len(segs[seg]) {
			start -= len(segs[seg])
			seg++
		}
	}

	return res
}

func (m *Model)renderStatusBar() string {
	cont := m.marks.Contains(m.cursorLine)
	pred := m.marks.Predecessor(m.cursorLine)
	succ := m.marks.Successor(m.cursorLine)
	marks := ""
	if pred != nil {
		marks = marks + fmt.Sprintf("↑%d", m.cursorLine - *pred)
	}
	if cont {
		marks = marks + "◎"
	}
	if succ != nil {
		marks = marks + fmt.Sprintf("↓%d", *succ - m.cursorLine)
	}

	lines := ""
	if m.eof {
		lines = fmt.Sprintf("L: %d/%d ", m.cursorLine + 1, len(m.lines))
	} else {
		lines = fmt.Sprintf("L: %d/∞ ", m.cursorLine + 1)
	}

	rest := fmt.Sprintf("%-*s",
		m.width - (utf8.RuneCountInString(lines) + utf8.RuneCountInString(marks)), "")

	st1 := m.statusStyle.Render(lines)
	st1 += m.markStyle.Render(marks)
	st1 += m.statusStyle.Render(rest)

	st2 := m.statusStyle.Render(" ")

	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern == "" {
			continue
		}

		pred = m.searchHits[i].Predecessor(m.cursorLine)
		succ = m.searchHits[i].Successor(m.cursorLine)

		sh := ""
		if pred != nil {
			sh = sh + fmt.Sprintf("↑%d", m.cursorLine - *pred)
		}
		if succ != nil {
			sh = sh + fmt.Sprintf("↓%d", *succ - m.cursorLine)
		}
		if m.searches[i].eof {
			sh = sh + "✓"
		}

		st2 += m.searchStyles[i].Render(sh)
	}

	st2 += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))

	return "\n" + st1 + "\n" + st2
}

func (m *Model) renderSearchBar() string {
	res := ""

	text := "  <Tab>: (regex/plain)  Ctrl+h/Ctrl+l or F3/F4: Next/Prev"
	res += m.statusStyle.Render(fmt.Sprintf("\n%-*s", m.width, text))

	prompt := fmt.Sprintf("%d: ", m.search + 1)

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
    go func() {
        scanner := bufio.NewScanner(m.file)
        for scanner.Scan() {
            m.input <- scanner.Text()
        }
        close(m.input) // Close the channel once done reading
    }()

	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	log.Println("Got request", msg)

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
					m.searchHits[m.search] = NewAVLTree[int](intComp)
				}
				m.cursorLine = m.findNext(m.search, math.MaxInt)
			} else if k == "tab" {
				m.newSearch.regex = !m.newSearch.regex
			} else if k == "esc" || k == "ctrl+c" {
				m.textbox.SetValue(m.searches[m.search].pattern)
				m.activeSearchTextbox = false
				m.textbox.Blur()
			} else if k == "ctrl+l" || k == "f4" {
				if m.newSearch.pattern != m.searches[m.search].pattern ||
					m.newSearch.regex != m.searches[m.search].regex {
					m.searches[m.search] = m.newSearch
					m.searchHits[m.search] = NewAVLTree[int](intComp)
				}
				if m.search < len(m.searches) - 1 {
					m.search++
				}
			} else if k == "ctrl+h" || k == "f3" {
				if m.newSearch.pattern != m.searches[m.search].pattern ||
					m.newSearch.regex != m.searches[m.search].regex {
					m.searches[m.search] = m.newSearch
					m.searchHits[m.search] = NewAVLTree[int](intComp)
				}
				if m.search > 0 {
					m.search--
				}
			} else {
				m.textbox, cmd = m.textbox.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else {
			if k == "q" || k == "ctrl+c" {
				return m, tea.Quit
			}

			if k == "up" || k == "k" {
				m.onKeyUp()
			}

			if k == "down" || k == "j" {
				m.onKeyDown()
			}

			if k == "g" {
				m.top = Position{ 0, 0 }
				m.cursorLine = 0
			}

			if k == "G" {
				m.onDocumentEnd()
			}

			if k == "pgup" {
				m.onPageUp()
			}

			if k == "pgdown" {
				m.onPageDown()
			}

			if k == "f" {
				m.onHalfPageDown()
			}

			if k == "b" {
				m.onHalfPageUp()
			}

			if k == " " {
				m.onToggleMark()
			}

			if k == "enter" {
				m.centerCursor()
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
				m.cursorLine = m.findNext(m.search, math.MaxInt)
			}

			if k == "N" {
				m.cursorLine = m.findPrev(m.search)
			}

			if search, ok := searchForwardRunes[k]; ok {
				if m.searches[search].pattern != "" {
					m.cursorLine = m.findNext(search, math.MaxInt)
				}
			}

			if search, ok := searchBackwardRunes[k]; ok {
				if m.searches[search].pattern != "" {
					m.cursorLine = m.findPrev(search)
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
	}

	m.normalizeView()

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	h := m.height
	w := m.width
	res := ""
	renderedLines := 0

	current := m.top
	desc := ""

	log.Printf("Rendering the view from line %v, cursor %d, dimensions %dx%d\n",
		current, m.cursorLine, w, h)

	m.populate(current.Line + h)

	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern != "" {
			m.findNext(i, current.Line + h)
		}
	}

	for h > 0 {
		if current.Line >= len(m.lines) {
			break
		}

		line := m.lines[current.Line]

		var cegs []Cegment
		if m.cursorLine == current.Line {
			cegs = m.cegmentize(line, &m.cursorStyle)
		} else {
			if m.marks.Contains(current.Line) {
				cegs = m.cegmentize(line, &m.markStyle)
			} else {
				cegs = m.cegmentize(line, &m.defaultStyle)
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
			desc = desc + fmt.Sprintf("%v ", current)
			h -= 1
		}

		current.Line += 1
		current.Segment = 0
	}

	for renderedLines < m.height {
		res += fmt.Sprintf("%-*s\n", w, " ")
		renderedLines++
	}

	if m.activeSearchTextbox {
		res += m.renderSearchBar()
	} else {
		res += m.renderStatusBar()
	}

	log.Printf("Rendered: %s, %d lines\n", strings.Trim(desc, " "), renderedLines)

	return res
}

