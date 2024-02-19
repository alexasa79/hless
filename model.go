package main

import (
	"bufio"
	"fmt"
	"io"
	"log"

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

	top Position
	cursorLine int

	pagerWidth int
	pagerHeight int

	cursorStyle lipgloss.Style
	statusStyle lipgloss.Style
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

		cursorStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#ffda45")),
		statusStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#fff")),
	}
}

func (m *Model) HasNext(pos Position) bool {
	m.populate(pos.Line)
	segs := m.Segmentize(m.lines[pos.Line])
	if pos.Segment < len(segs) - 1 {
		return true
	}
	if pos.Line < len(m.lines) - 1 {
		return true
	}
	return false
}

func (m *Model) Next(pos Position) Position {
	m.populate(pos.Line)
	segs := m.Segmentize(m.lines[pos.Line])
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

func (m *Model) HasPrev(pos Position) bool {
	if pos.Line == 0 && pos.Segment == 0 {
		return false
	}
	return true
}

func (m *Model) Prev(pos Position) Position {
	if pos.Segment > 0 {
		pos.Segment--
		return pos
	}

	if pos.Line > 0 {
		pos.Line--
		segs := m.Segmentize(m.lines[pos.Line])
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
func (m *Model) BottomLine() (int, int) {
	pos := Position { m.top.Line, m.top.Segment }
	for h := 0; m.HasNext(pos) && h < m.pagerHeight - 1; h++ {
		pos = m.Next(pos)
	}
	return pos.Line, pos.Segment
}

func (m *Model) ScrollOneSegmentDown() {
	m.top.Segment += 1
	segs := m.Segmentize(m.lines[m.top.Line])
	if m.top.Segment >= len(segs) {
		if m.top.Line < len(m.lines) - 1 {
			m.top.Line += 1
			m.top.Segment = 0
		}
	}
}

func (m *Model) OnKeyUp() {
	if m.cursorLine != m.top.Line {
		m.cursorLine -= 1
	} else {
		if m.top.Segment > 0 {
			m.top.Segment -= 1
		} else {
			if m.top.Line > 0 {
				m.top.Line -= 1
				segs := m.Segmentize(m.lines[m.top.Line])
				m.top.Segment = len(segs) - 1
				m.cursorLine = m.top.Line
			}
		}
	}
}

func (m *Model) OnKeyDown() {
	h := m.pagerHeight

	m.populate(m.cursorLine + h)

	if m.cursorLine < len(m.lines) - 1 {
		bottomLine, bottomSegment := m.BottomLine()
		log.Printf("Scrolling down. Top line is %v, cursor %d, bottom %d(%d)",
			m.top, m.cursorLine, bottomLine, bottomSegment)
		if m.cursorLine == bottomLine {
			m.ScrollOneSegmentDown()
			bottomLine, bottomSegment = m.BottomLine()
			log.Printf("New bottom %d(%d)\n", bottomLine, bottomSegment)
			if m.cursorLine < bottomLine {
				m.cursorLine += 1
			}
		} else {
			m.cursorLine += 1
		}
	} else {
		bottomLine, bottomSegment := m.BottomLine()
		segs := m.Segmentize(m.lines[bottomLine])
		if (bottomSegment < len(segs) - 1) {
			m.ScrollOneSegmentDown()
		}
	}
}

func (m *Model) OnDocumentEnd() {
	h := m.pagerHeight

	m.populate(2147483647)

	m.cursorLine = len(m.lines) - 1
	line := len(m.lines) - 1

	var segs []string
	for {
		segs = m.Segmentize(m.lines[line])
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

func (m *Model) OnPageDown() {
	h := m.pagerHeight - 2
	for h > 0 {
		segs := m.Segmentize(m.lines[m.cursorLine])
		h -= len(segs)
		if m.cursorLine >= len(m.lines) - 1 {
			break
		}
		m.cursorLine++
	}

	m.NormalizeView()
}

/// Given current position of the cursor, make sure it is visible.
/// This is needed in cases when change of terminal's size causes the cursor
/// to become invisible (usually by being below bottom line).
func (m *Model) NormalizeView() {
	if !m.ready {
		return
	}

	m.populate(m.top.Line + m.pagerHeight)

	if m.top.Line > m.cursorLine {
		log.Printf("Cursor %d appears to be before top line %d", m.cursorLine, m.top.Line)
		m.top.Line = m.cursorLine
		m.top.Segment = 0
		log.Printf("Fixed top line to %v", m.top)
		return
	}

	bottomLine, _ := m.BottomLine()
	if m.cursorLine > bottomLine {
		log.Printf("Cursor %d appears to be after bottom line %d", m.cursorLine, bottomLine)
		pos := Position { m.cursorLine, len(m.Segmentize(m.lines[m.cursorLine])) - 1 }
		for h := m.pagerHeight; m.HasPrev(pos) && h > 1; h-- {
			pos = m.Prev(pos)
		}
		m.top = pos
		log.Printf("Fixed top line to %v", m.top)
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
			m.OnKeyUp()
		}

		if k == "down" || k == "j" {
			m.OnKeyDown()
		}

		if k == "g" {
			m.top = Position{ 0, 0 }
			m.cursorLine = 0
		}

		if k == "G" {
			m.OnDocumentEnd()
		}

		if k == "pgdown" {
			m.OnPageDown()
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.pagerWidth = msg.Width
			m.pagerHeight = msg.Height - 2
			m.ready = true
		} else {
			m.pagerWidth = msg.Width
			m.pagerHeight = msg.Height - 2
		}
	}

	m.NormalizeView()

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

func (m *Model)Segmentize(line string) []string {
	res := make([]string, 0)
	for line != "" {
		l := min(m.pagerWidth, len(line))
		seg := line[:l]
		if len(seg) < m.pagerWidth {
			seg = fmt.Sprintf("%-*s", m.pagerWidth, seg)
		}
		res = append(res, seg)
		line = line[l:]
	}
	return res
}

func (m *Model) View() string {
	h := m.pagerHeight
	w := m.pagerWidth
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
		segs := m.Segmentize(line)

		for i, seg := range segs[current.Segment:] {
			if (h == 0) {
				break
			}

			if current.Line == m.cursorLine {
				seg = m.cursorStyle.Render(seg)
				desc = desc + "*"
			}
			if (res == "") {
				res = seg
			} else {
				res = res + "\n" + seg
			}
			renderedLines++
			desc = desc + fmt.Sprintf("%d(%d) ", current.Line, i)
			h -= 1
		}

		current.Line += 1
		current.Segment = 0
	}

	for renderedLines < m.pagerHeight {
		res += fmt.Sprintf("%-*s\n", w, " ")
		renderedLines++
	}

	if m.eof {
		st1 := fmt.Sprintf("L: %d/%d", m.cursorLine + 1, len(m.lines))
		st1 = fmt.Sprintf("%-*s", w, st1)
		res += "\n" + m.statusStyle.Render(st1)

		st2 := fmt.Sprintf("%-*s", w, "")
		res += "\n" + m.statusStyle.Render(st2)
	} else {
		st1 := fmt.Sprintf("L: %d/∞", m.cursorLine + 1)
		st1 = fmt.Sprintf("%-*s", w, st1)
		res += "\n" + m.statusStyle.Render(st1)

		st2 := fmt.Sprintf("%-*s", w, "")
		res += "\n" + m.statusStyle.Render(st2)
	}

	log.Printf("Rendered: %s, %d lines\n", desc, renderedLines)

	return res
}