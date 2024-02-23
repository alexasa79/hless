package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Position struct {
	Line int
	Segment int
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

	cursorStyle lipgloss.Style
	statusStyle lipgloss.Style
	markStyle lipgloss.Style
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
	return &Model {
		ready: false,

		file: f,
		input: make(chan string),
		lines: make([]string, 0),
		eof: false,

		top: Position{ 0, 0 },
		cursorLine: 0,

		marks: NewAVLTree[int](intComp),

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
	}
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

func (m *Model) onToggleMark() {
	log.Printf("Toggling mark at position %d\n", m.cursorLine)
	if m.marks.Contains(m.cursorLine) {
		m.marks.Remove(m.cursorLine)
	} else {
		m.marks.Insert(m.cursorLine)
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	log.Println("Got request", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

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

	return m, cmd
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

	st2 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))

	return "\n" + st1 + "\n" + st2
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

	for h > 0 {
		if current.Line >= len(m.lines) {
			break
		}

		line := m.lines[current.Line]
		segs := m.segmentize(line)

		for _, seg := range segs[current.Segment:] {
			if (h == 0) {
				break
			}

			if current.Line == m.cursorLine {
				seg = m.cursorStyle.Render(seg)
				desc = desc + "*"
			} else {
				if m.marks.Contains(current.Line) {
					seg = m.markStyle.Render(seg)
				}
			}
			if (res == "") {
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

	res += m.renderStatusBar()

	log.Printf("Rendered: %s, %d lines\n", strings.Trim(desc, " "), renderedLines)

	return res
}