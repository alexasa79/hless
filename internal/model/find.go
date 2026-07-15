package model

import (
	"strings"

	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) findFastInner(data []byte, search int, startOffset int64, first bool) (int64, bool) {
	var index int
	if m.searches[search].regex {
		matches := m.searches[search].re.FindAllIndex(data, -1)
		if matches != nil {
			if first {
				index = matches[0][0]
			} else {
				index = matches[len(matches)-1][0]
			}
		} else {
			index = -1
		}
	} else {
		if first {
			index = strings.Index(string(data), m.searches[search].pattern)
		} else {
			index = strings.LastIndex(string(data), m.searches[search].pattern)
		}
	}

	if index < 0 {
		SL.Trace("Pattern not found")
		return -1, false
	} else {
		SL.Trace("Found a match at offset %d (%d+%d)", startOffset+int64(index), startOffset, index)
		return int64(index) + startOffset, true
	}
}

func (m *Model) findNextWithProgress(search int) tea.Cmd {
	lastProgress := float64(0)
	currentOffset := m.cursorLine.Offset + m.lines.RawLength(m.cursorLine)
	// Overlap size between two buffers, in case the match is between two 10 MiB buffers.
	// Set later to 64 KiB
	shift := int64(0)

	SL.Info("Searching forward from %v. Looking for %v", m.cursorLine, m.searches[search])

	return func() tea.Msg {
		SL.Debug("Callback called for %v", currentOffset)

		if currentOffset >= m.buffer.Size() || m.searches[search].pattern == "" {
			return ProgressReport{
				progress:    1.0,
				interrupted: true,
			}
		}

		var searchLine *lines.Line
		var interrupted bool
		var prog float64

		batchSize := int64(10485760) // 10 MiB
		to := utils.Min(currentOffset+batchSize, m.buffer.Size())
		SL.Trace("Calling FastFind for range [%d, %d)", currentOffset-shift, to)
		searchLine = m.lines.FastFind(currentOffset-shift, to, true,
			func(data []byte, offset int64) (int64, bool) {
				return m.findFastInner(data, search, offset, true)
			})

		select {
		case interrupted = <-m.progressInterrupt:
			SL.Debug("Search interrupted")
		default:
		}

		if searchLine == nil {
			currentOffset = to
			if currentOffset >= m.buffer.Size() {
				prog = 1.0
				interrupted = true
			} else {
				prog = float64(currentOffset) / float64(m.buffer.Size())
			}
		} else {
			SL.Debug("Setting cursor to %v", searchLine)
			m.cursorLine = searchLine
			interrupted = true
		}

		shift = 65536

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		res := ProgressReport{
			progress:    prog,
			interrupted: interrupted,
		}

		SL.Debug("Returning progress report %v", res)

		return res
	}
}

func (m *Model) findPrevWithProgress(search int) tea.Cmd {
	lastProgress := float64(0)
	currentOffset := m.cursorLine.Offset
	// Overlap size between two buffers, in case the match is between two 10 MiB buffers.
	// Set later to 64 KiB
	shift := int64(0)

	SL.Info("Searching backwards from %v. Searching for %v", m.cursorLine, m.searches[search])

	return func() tea.Msg {
		SL.Debug("Callback called for %v", currentOffset)

		if currentOffset >= m.buffer.Size() || m.searches[search].pattern == "" {
			return ProgressReport{
				progress:    0,
				interrupted: true,
			}
		}

		var searchLine *lines.Line
		var interrupted bool
		var prog float64

		batchSize := int64(10485760) // 10 MiB
		from := utils.Max(0, currentOffset-batchSize)
		SL.Trace("Calling FastFind for range [%d, %d)", from, currentOffset+shift)
		searchLine = m.lines.FastFind(from, currentOffset+shift, false,
			func(data []byte, offset int64) (int64, bool) {
				return m.findFastInner(data, search, offset, false)
			})

		select {
		case interrupted = <-m.progressInterrupt:
			SL.Debug("Search interrupted")
		default:
		}

		if searchLine == nil {
			currentOffset = from
			if currentOffset == 0 {
				prog = 1.0
				interrupted = true
			} else {
				prog = 1.0 - (float64(currentOffset) / float64(m.buffer.Size()))
			}
		} else {
			SL.Debug("Setting cursor to %v", searchLine)
			m.cursorLine = searchLine
			interrupted = true
		}

		shift = 65536

		prog = utils.Max(prog, lastProgress)
		lastProgress = prog

		res := ProgressReport{
			progress:    prog,
			interrupted: interrupted,
		}

		SL.Debug("Returning progress report %v", res)

		return res
	}
}
