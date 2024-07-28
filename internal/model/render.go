package model

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
)

func (m *Model) renderShowBar() string {
	res := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, " "))

	prompt := "Show (regex)❱ "
	m.textbox.Width = m.width
	m.textbox.Prompt = prompt
	m.textbox.PromptStyle = m.promptStyle
	m.textbox.TextStyle = m.editorStyle
	res += "\n" + m.textbox.View()

	return res
}

func (m *Model) renderHideBar() string {
	res := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, " "))

	prompt := "Hide (regex)❱ "
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
		prompt += "(regex)❱ "
	} else {
		prompt += "(plain)❱ "
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
	m.textbox.Prompt = "❱ "
	m.textbox.PromptStyle = m.promptStyle
	m.textbox.TextStyle = m.editorStyle

	res += "\n" + m.textbox.View()

	return res
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

func (m *Model) renderStatusBar() string {
	VL.Debug("Rendering status bar with cursor %v", m.cursorLine)

	if m.cursorLine == nil {
		st1 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, "EMPTY BUFFER"))
		st2 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))
		return st1 + "\n" + st2
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
			if current.Line == m.cursorLine {
				desc = desc + fmt.Sprintf("*%d ", current.Line.Offset)
			} else {
				desc = desc + fmt.Sprintf("%d ", current.Line.Offset)
			}
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
