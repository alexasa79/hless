package model

import (
	"fmt"
	"reflect"
	"time"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/jumpparser"
	"github.com/alexasa79/hless/internal/lines"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	viewCommand := m.viewCommand
	m.viewCommand = false

	ML.Info("Got request %v %v", reflect.TypeOf(msg), msg)

	m.lines.Refresh()

	cmds := make([]tea.Cmd, 0)

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
		} else if m.inventoryActive {
			if k == "q" {
				m.inventoryActive = false
				break
			}

			m.inventoryViewport, cmd = m.inventoryViewport.Update(msg)
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

					if m.searches[m.search].pattern == "" {
						m.search = m.prevSearch
					} else {
						if m.activeSearchDirection == SEARCH_DOWN {
							cmd = m.startProgress("Searching", m.findNextWithProgress(m.search))
							cmds = append(cmds, cmd)
						} else if m.activeSearchDirection == SEARCH_UP {
							cmd = m.startProgress("Searching", m.findPrevWithProgress(m.search))
							cmds = append(cmds, cmd)
						}
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
					m.lines.Refresh()
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

				if k == "M" {
					m.onClearMark()
				}

				if k == "." {
					m.nextMark()
				}

				if k == "," {
					m.prevMark()
				}

				if k == "e" {
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
					m.helpViewport.SetContent(m.renderHelp())
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

				if k == "shift+left" {
					m.onExpandSelectionLeft()
				}

				if k == "right" {
					m.onKeyRight()
				}

				if k == "shift+right" {
					m.onExpandSelectionRight()
				}

				if k == "a" {
					if viewCommand {
						m.hideAfterCursor()
					} else {
						for i := 0; i < len(m.searches); i++ {
							if m.searches[m.search].pattern != "" {
								m.search = (m.search + 1) % len(m.searches)
							} else {
								break
							}
						}

						if m.searches[m.search].pattern == "" {
							m.onAddCurrentWord()
						} else {
							m.setError("Out of search slots")
						}
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
					if viewCommand {
						m.onAddCurrentWordShowPattern()
					} else {
						m.onAddCurrentWord()
					}
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

				if k == "i" {
					m.inventoryActive = true
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

		m.inventoryViewport, cmd = m.inventoryViewport.Update(msg)
		cmds = append(cmds, cmd)

	case ProgressReport:
		if !msg.interrupted {
			m.currentProgress = msg.progress
			cmds = append(cmds, m.progressCmd)
		} else {
			m.inProgress = false
			m.progressInterrupt = nil
		}

	case lines.BufferRefreshSignal:
		cmds = append(cmds, m.bufferRefresher)
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

	if m.ready && m.top.Line == nil {
		m.top.Line = m.lines.First()
		m.top.Segment = 0
		m.cursorLine = m.top.Line
		ML.Trace("Missing top line. Updated to %v", m.top)
	}

	m.normalizeView()

	if m.prevCursorLine != m.cursorLine {
		m.prevCursorLine = m.cursorLine
		m.selectionStart = -1
		m.selectionEnd = -1
	}

	ML.Debug("Returning commands %v", cmds)

	return m, tea.Batch(cmds...)
}
