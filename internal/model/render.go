package model

import (
	"fmt"
	"strings"
	"time"

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

	text := "NUM for absolute line, [+|-]NUM for relative. [+|-]NUM[us|ms|s|m|h] for relative time"
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

func (m *Model) renderStatusHeader(cts *time.Time) string {
	res := fmt.Sprintf("Line %s @%d ", m.cursorLine.Name(), m.cursorLine.Offset)
	if cts != nil {
		res += cts.String()
	}
	return m.statusStyle.Render(res)
}

func (m *Model) renderStatusMarks(cts *time.Time) string {
	rawMarks := ""
	renderedMarks := ""

	for i := 0; i < len(m.markStyle); i++ {
		mls := MarkLine{line: m.cursorLine}
		ml := m.marks.Get(&mls)
		if ml != nil && (*ml).style != i {
			ml = nil
		}
		pred := m.marks.Predecessor(&mls)
		for pred != nil && (*pred).style != i {
			pred = m.marks.Predecessor(*pred)
		}
		succ := m.marks.Successor(&mls)
		for succ != nil && (*succ).style != i {
			succ = m.marks.Successor(*succ)
		}

		ms := make([]string, 0)
		if pred != nil {
			ms = append(ms, fmt.Sprintf("↑%s", lines.Distance((*pred).line, m.cursorLine)))
			ttt := utils.ExtractTimestamp(m.lines.Text((*pred).line))
			if cts != nil && ttt != nil {
				ms = append(ms, (*cts).Sub(*ttt).String())
			}
		}
		if ml != nil {
			ms = append(ms, "†")
		}
		if succ != nil {
			ms = append(ms, fmt.Sprintf("%s↓", lines.Distance((*succ).line, m.cursorLine)))
			ttt := utils.ExtractTimestamp(m.lines.Text((*succ).line))
			if cts != nil && ttt != nil {
				ms = append(ms, (*ttt).Sub(*cts).String())
			}
		}

		rawMarks += strings.Join(ms, " ")
		renderedMarks += m.markStyle[i].Render(strings.Join(ms, " "))
	}

	return renderedMarks
}

func (m *Model) renderStatusSearch(cts *time.Time) string {
	res := m.statusStyle.Render("")

	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern == "" {
			continue
		}

		// var pred *lines.Line = nil
		predSearchHit := m.searchHits[i].Predecessor(m.cursorLine)
		if predSearchHit != nil && !m.lines.Visible(*predSearchHit) {
			predSearchHit = nil
		}

		succSearchHit := m.searchHits[i].Successor(m.cursorLine)
		if succSearchHit != nil && !m.lines.Visible(*succSearchHit) {
			succSearchHit = nil
		}

		sh := make([]string, 0)
		if predSearchHit != nil {
			sh = append(sh, fmt.Sprintf("↑%s", lines.Distance(*predSearchHit, m.cursorLine)))
			ttt := utils.ExtractTimestamp(m.lines.Text(*predSearchHit))
			if ttt != nil && cts != nil {
				sh = append(sh, (cts.Sub(*ttt).String()))
			}
		}

		if succSearchHit != nil {
			sh = append(sh, fmt.Sprintf("%s↓", lines.Distance(*succSearchHit, m.cursorLine)))
			ttt := utils.ExtractTimestamp(m.lines.Text(*succSearchHit))
			if ttt != nil && cts != nil {
				sh = append(sh, (ttt.Sub(*cts).String()))
			}
		}

		sh = append(sh, fmt.Sprintf("◎%d", m.searchHits[i].Count()))

		text := strings.Join(sh, " ")

		if i == m.search {
			m.searchStyles[i].Bold(true)
		}

		res += m.searchStyles[i].Render(text)
		m.searchStyles[i].Bold(false)

		res += m.statusStyle.Render(" ")
	}

	return res
}

func (m *Model) renderStatusBar() string {
	VL.Debug("Rendering status bar with cursor %v", m.cursorLine)

	if m.cursorLine == nil {
		st1 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, "EMPTY BUFFER"))
		st2 := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))
		return st1 + "\n" + st2
	}

	if m.viewCommand {
		st1 := " VIEW options"
		res := m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, st1))

		res += "\n"

		st2 := " b/a - hide before/after, s/h - show/hide pattern, c - clear, A - show current word"
		res += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, st2))

		return m.statusStyle.Render(res)
	}

	cts := utils.ExtractTimestamp(m.lines.Text(m.cursorLine))

	st1 := m.renderStatusHeader(cts)
	marks := m.renderStatusMarks(cts)

	if marks != "" {
		st1 += m.statusStyle.Render(" ") + marks
	}

	st1 += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))

	st2 := ""
	if m.activeError {
		st2 += m.errorStyle.Render(fmt.Sprintf("%-*s", m.width, m.errorText))
	} else {
		st2 += m.renderStatusSearch(cts)
	}

	st2 += m.statusStyle.Render(fmt.Sprintf("%-*s", m.width, ""))

	return st1 + "\n" + st2
}

func (m *Model) overlayHelp(onto string) string {
	w := 80
	h := int(max(25, float64(m.height)*0.8))
	x := (m.width - w) / 2
	y := (m.height - h) / 2

	m.helpViewport.Width = w
	m.helpViewport.Height = h

	help := m.helpViewport.View()
	return PlaceOverlay(x, y, help, onto)
}

func (m *Model) overlayInventory(onto string) string {
	w := 80
	h := 25
	x := (m.width - w) / 2
	y := (m.height - h) / 2

	text := " Build " + m.buildTime + " " + m.gitrev + "\n"

	text += "\n " + m.textHeader1Style.Render("# File information") + "\n\n"

	text += " " + m.boldTextStyle.Render("File name  : ")
	text += fmt.Sprintf(" %s\n", m.buffer.Name())

	text += " " + m.boldTextStyle.Render("Buffer size: ")
	text += fmt.Sprintf(" %d bytes\n", m.buffer.Size())

	text += "\n " + m.textHeader1Style.Render("# Search patterns") + "\n\n"
	for i := 0; i < len(m.searches); i++ {
		if m.searches[i].pattern != "" {
			regex := "plain"
			if m.searches[i].regex {
				regex = "regex"
			}
			text += " " + m.boldTextStyle.Render(fmt.Sprintf("%d: ", i+1))
			text += m.searchStyles[i].Render(
				fmt.Sprintf("(%s) %s", regex, m.searches[i].pattern)) + "\n"
		}
	}

	text += "\n " + m.textHeader1Style.Render("# Show/hide patterns") + "\n\n"
	text += " " + m.boldTextStyle.Render("Show: ")
	text += m.showPattern + "\n"
	text += " " + m.boldTextStyle.Render("Hide: ")
	text += m.hidePattern + "\n"

	m.inventoryViewport.SetContent(text)

	m.inventoryViewport.Width = w
	m.inventoryViewport.Height = h

	inventory := m.inventoryViewport.View()
	return PlaceOverlay(x, y, inventory, onto)
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
			cegs = m.cegmentize(current.Line, &m.cursorStyle, m.selectionStart, m.selectionEnd, &m.currentWordStyle, true)
		} else {
			mls := MarkLine{line: current.Line}
			ml := m.marks.Get(&mls)
			if ml != nil {
				cegs = m.cegmentize(current.Line, &m.markStyle[(*ml).style], -1, -1, &m.currentWordStyle, true)
			} else {
				cegs = m.cegmentize(current.Line, &m.defaultStyle, -1, -1, &m.currentWordStyle, false)
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

	if m.helpActive {
		res = m.overlayHelp(res)
	} else if m.inventoryActive {
		res = m.overlayInventory(res)
	}

	return res
}
