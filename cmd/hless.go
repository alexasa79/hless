package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	ready bool

	file *os.File
	input chan string
	lines []string
	eof bool

	topLine int
	topSegment int

	cursorLine int

	cursorStyle lipgloss.Style

	viewport viewport.Model
}

func initialModel(f *os.File) *model {
	return &model {
		ready: false,

		file: f,
		input: make(chan string),
		lines: make([]string, 0),
		eof: false,

		topLine: 0,
		topSegment: 0,
		cursorLine: 0,

		cursorStyle: lipgloss.
			NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#ffda45")),
	}
}

func (m *model) Init() tea.Cmd {
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
func (m *model) bottomLine() (int, int) {
	w := m.viewport.Width
	h := m.viewport.Height
	ts := m.topSegment
	bottomLine := m.topLine
	bottomSegment := 0
	for h > 0 {
		segs := segmentize(m.lines[bottomLine], w)
		bottomSegment = min(len(segs[ts:]), ts+h) - 1
		h -= len(segs[ts:])
		ts = 0
		if h > 0 {
			bottomLine++
		}
	}
	return bottomLine, bottomSegment
}

func (m *model) scrollOneSegmentDown() {
	m.topSegment += 1
	segs := segmentize(m.lines[m.topLine], m.viewport.Width)
	if m.topSegment >= len(segs) {
		if m.topLine < len(m.lines) - 1 {
			m.topLine += 1
			m.topSegment = 0
		}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	w := m.viewport.Width

	log.Println("Got request", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()

		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}

		if k == "up" || k == "k" {
			if m.cursorLine != m.topLine {
				m.cursorLine -= 1
			} else {
				if m.topSegment > 0 {
					m.topSegment -= 1
				} else {
					if m.topLine > 0 {
						m.topLine -= 1
						segs := segmentize(m.lines[m.topLine], w)
						m.topSegment = len(segs) - 1
						m.cursorLine = m.topLine
					}
				}
			}
		}

		if k == "down" || k == "j" {
			if m.cursorLine < len(m.lines) - 1 {
				bottomLine, bottomSegment := m.bottomLine()
				log.Printf("Scrolling down. Top line is %d(%d), cursor %d, bottom %d(%d)",
					m.topLine, m.topSegment, m.cursorLine, bottomLine, bottomSegment)
				if m.cursorLine == bottomLine {
					m.scrollOneSegmentDown()
					bottomLine, _ = m.bottomLine()
					if m.cursorLine < bottomLine {
						m.cursorLine += 1
					}
				} else {
					m.cursorLine += 1
				}
			} else {
				bottomLine, bottomSegment := m.bottomLine()
				segs := segmentize(m.lines[bottomLine], w)
				if (bottomSegment < len(segs) - 1) {
					m.scrollOneSegmentDown()
				}
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m *model) populate(n int) {
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

func segmentize(line string, w int) []string {
	res := make([]string, 0)
	for line != "" {
		l := min(w, len(line))
		seg := line[:l]
		if len(seg) < w {
			seg = fmt.Sprintf("%-*s", w, seg)
		}
		res = append(res, seg)
		line = line[l:]
	}
	return res
}

func (m *model) View() string {
	h := m.viewport.Height
	w := m.viewport.Width
	res := ""

	currentLine := m.topLine
	currentSegment := m.topSegment
	desc := ""

	log.Printf("Rendering the view from line %d(%d), cursor %d, dimensions %dx%d\n",
		currentLine, currentSegment, m.cursorLine, w, h)

	m.populate(currentLine + h)

	for h > 0 {
		if currentLine >= len(m.lines) {
			break
		}

		line := m.lines[currentLine]
		segs := segmentize(line, w)

		for i, seg := range segs[currentSegment:] {
			if (h == 0) {
				break
			}

			if currentLine == m.cursorLine {
				seg = m.cursorStyle.Render(seg)
				desc = desc + "*"
			}
			if (res == "") {
				res = seg
			} else {
				res = res + "\n" + seg
			}
			desc = desc + fmt.Sprintf("%d(%d) ", currentLine, i)
			h -= 1
		}

		currentLine += 1
		currentSegment = 0
	}

	log.Printf("Rendered: %s\n", desc)

	return res
}

func parseArguments() (*string, error) {
	if (len(os.Args) != 2) {
		return nil, errors.New("missing input file")
	}
	return &os.Args[1], nil
}

func main() {
	file, err := parseArguments()
	if err != nil {
		fmt.Println("Error parsing arguments:", err)
		os.Exit(1)
	}
	in, err := os.Open(*file)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		os.Exit(1)
	}
	defer in.Close()

	f, err := tea.LogToFile("hless.log", "")
	if err != nil {
		fmt.Println("Error opening debug log:", err)
		os.Exit(1)
	}
	defer f.Close()

	log.Println("-----------------------------------------------------------")

    p := tea.NewProgram(initialModel(in), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error rendering the TUI: %v", err)
        os.Exit(1)
    }
}