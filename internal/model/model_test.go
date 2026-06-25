package model

import (
	"strings"
	"testing"
	"time"

	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/utils"
)

func createModel(s string, w int, h int) *Model {
	b := lines.NewBufferFromString(s)
	m := InitialModel(b, "", "")
	m.width = w
	m.height = h
	m.ready = true
	m.Init()
	return m
}

func TestFormatStatusTimestampOmitsTimezone(t *testing.T) {
	t.Parallel()

	ts, err := time.Parse(time.RFC3339Nano, "2026-06-25T17:27:47.65811+00:00")
	if err != nil {
		t.Fatal(err)
	}

	got := formatStatusTimestamp(ts)
	want := "2026-06-25 17:27:47.65811"
	if got != want {
		t.Fatalf("formatStatusTimestamp() = %q, want %q", got, want)
	}
}

func (m *Model) verifyState(t *testing.T, line int64, segment int, cursor int64) {
	lines := make([]*lines.Line, 0)
	l := m.lines.First()
	for l != nil {
		lines = append(lines, l)
		l = m.lines.Next(l)
	}

	if m.top.Line != lines[line] || m.top.Segment != segment || m.cursorLine != lines[cursor] {
		t.Fatalf("Unexpected state: top=%v, cursor=%v. Expected %d(%d) and %v\n",
			m.top, m.cursorLine, line, segment, cursor)
	}
}

func TestDown1(t *testing.T) {
	t.Parallel()
	m := createModel("012345678\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown() // showing 12
	m.verifyState(t, 0, 1, 0)

	m.onKeyDown() // 23
	m.onKeyDown() // 34
	m.onKeyDown() // 45
	m.onKeyDown() // 56
	m.onKeyDown() // 67
	m.onKeyDown() // 78
	m.verifyState(t, 0, 7, 0)

	m.onKeyDown() // 78
	m.verifyState(t, 0, 7, 0)
}

func TestDown2(t *testing.T) {
	t.Parallel()
	m := createModel("01\n23\n45\n67\n78\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown() // showing 1\n2
	m.verifyState(t, 0, 1, 1)
}

func TestUp(t *testing.T) {
	t.Parallel()
	m := createModel("012345678\n", 1, 2)
	m.verifyState(t, 0, 0, 0)
	m.onKeyDown()
	m.verifyState(t, 0, 1, 0)
	m.onKeyUp()
	m.verifyState(t, 0, 0, 0)
}

func TestPageDown(t *testing.T) {
	t.Parallel()
	m := createModel("01\n23\n45\n67\n78\n", 1, 2)
	m.onPageDown()
	m.verifyState(t, 0, 0, 0)
	m.onPageDown()
	m.verifyState(t, 0, 0, 0)
}

func TestRenderLineWithRelativeTimestamp(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)

	if !strings.Contains(rl.text, "2026-06-25 14:37:50.721999891 +0.000000000 0x10002") {
		t.Fatalf("rendered line %q does not contain relative timestamp", rl.text)
	}
	if rl.raw != raw {
		t.Fatalf("raw line changed: got %q, want %q", rl.raw, raw)
	}
}

func TestRelativeTimestampSearchRangeSpansInsertedColumn(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)

	matchStart := strings.Index(raw, "14:37:50.721999891 0x10002")
	if matchStart == -1 {
		t.Fatal("test fixture does not contain search text")
	}
	rawRange := SearchResult{start: matchStart, end: matchStart + len("14:37:50.721999891 0x10002")}
	displayRange := rl.displayRange(rawRange)

	got := rl.text[displayRange.start:displayRange.end]
	want := "14:37:50.721999891 +0.000000000 0x10002"
	if got != want {
		t.Fatalf("display match = %q, want %q", got, want)
	}
}

func TestRelativeTimestampPrecisionMatchesZeroLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		zero string
		line string
		want string
	}{
		{
			name: "milliseconds",
			zero: "2026-06-25 14:37:50.721 zero",
			line: "2026-06-25 14:37:50.722 next",
			want: "2026-06-25 14:37:50.722 +0.001 next",
		},
		{
			name: "microseconds",
			zero: "2026-06-25 14:37:50.721999 zero",
			line: "2026-06-25 14:37:50.722016 next",
			want: "2026-06-25 14:37:50.722016 +0.000017 next",
		},
		{
			name: "nanoseconds",
			zero: "2026-06-25 14:37:50.721999891 zero",
			line: "2026-06-25 14:37:50.722016601 next",
			want: "2026-06-25 14:37:50.722016601 +0.000016710 next",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createModel(tt.zero+"\n"+tt.line+"\n", 200, 5)
			m.showRelativeTimestamp()

			next := m.lines.Next(m.cursorLine)
			raw := m.lines.Text(next)
			ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
			rl := m.renderLine(raw, ts, timestampEnd)

			if rl.text != tt.want {
				t.Fatalf("rendered line = %q, want %q", rl.text, tt.want)
			}
		})
	}
}

func TestRelativeTimestampInsertedColumnUsesSearchStyle(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()
	m.searches[0] = Search{pattern: "14:37:50.721999891 0x10002"}

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)
	cegs := m.cegmentizeRendered(rl, &m.defaultStyle, -1, -1, &m.currentWordStyle, false)

	expectedStyle := m.searchStyles[0].Copy().Bold(true)
	for _, ceg := range cegs {
		for i, subsegment := range ceg.subsegments {
			if strings.Contains(subsegment, "+0.000000000") {
				got := ceg.style[i].Render(subsegment)
				want := expectedStyle.Render(subsegment)
				if got != want {
					t.Fatalf("relative timestamp column style mismatch for %q", subsegment)
				}
				return
			}
		}
	}

	t.Fatalf("rendered segments did not contain relative timestamp column: %+v", cegs)
}

func TestRelativeTimestampInsertedColumnUsesMetadataStyle(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)
	cegs := m.cegmentizeRendered(rl, &m.defaultStyle, -1, -1, &m.currentWordStyle, false)

	for _, ceg := range cegs {
		for i, subsegment := range ceg.subsegments {
			if strings.Contains(subsegment, "+0.000000000") {
				got := ceg.style[i].Render(subsegment)
				expectedStyle := m.relativeTimestampStyle(&m.defaultStyle)
				want := expectedStyle.Render(subsegment)
				if got != want {
					t.Fatalf("relative timestamp metadata style mismatch for %q", subsegment)
				}
				return
			}
		}
	}

	t.Fatalf("rendered segments did not contain relative timestamp column: %+v", cegs)
}

func TestRelativeTimestampInsertedColumnUsesCursorBackground(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)
	cegs := m.cegmentizeRendered(rl, &m.cursorStyle, -1, -1, &m.currentWordStyle, true)

	for _, ceg := range cegs {
		for i, subsegment := range ceg.subsegments {
			if strings.Contains(subsegment, "+0.000000000") {
				got := ceg.style[i].Render(subsegment)
				expectedStyle := m.relativeTimestampStyle(&m.cursorStyle)
				want := expectedStyle.Render(subsegment)
				if got != want {
					t.Fatalf("relative timestamp cursor style mismatch for %q", subsegment)
				}
				return
			}
		}
	}

	t.Fatalf("rendered segments did not contain relative timestamp column: %+v", cegs)
}

func TestRelativeTimestampInsertedColumnUsesSelectionStyle(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	ts, timestampEnd := utils.ExtractTimestampPrefix(raw)
	rl := m.renderLine(raw, ts, timestampEnd)

	m.selectionStart = strings.Index(raw, "14:37:50.721999891")
	if m.selectionStart == -1 {
		t.Fatal("test fixture does not contain selection start")
	}
	m.selectionEnd = strings.Index(raw, "0x10002") + len("0x10002")
	cegs := m.cegmentizeRendered(rl, &m.defaultStyle, m.selectionStart, m.selectionEnd, &m.currentWordStyle, false)

	for _, ceg := range cegs {
		for i, subsegment := range ceg.subsegments {
			if strings.Contains(subsegment, "+0.000000000") {
				got := ceg.style[i].Render(subsegment)
				want := m.currentWordStyle.Render(subsegment)
				if got != want {
					t.Fatalf("relative timestamp column selection style mismatch for %q", subsegment)
				}
				return
			}
		}
	}

	t.Fatalf("rendered segments did not contain relative timestamp column: %+v", cegs)
}

func TestAddSelectedTextAcrossRelativeTimestampUsesRawLine(t *testing.T) {
	t.Parallel()

	m := createModel("2026-06-25 14:37:50.721999891 0x10002\n", 200, 5)
	m.showRelativeTimestamp()

	raw := m.lines.Text(m.cursorLine)
	m.selectionStart = strings.Index(raw, "14:37:50.721999891")
	if m.selectionStart == -1 {
		t.Fatal("test fixture does not contain selection start")
	}
	m.selectionEnd = strings.Index(raw, "0x10002") + len("0x10002")

	m.onAddCurrentWord()

	want := "14:37:50.721999891 0x10002"
	if m.searches[m.search].pattern != want {
		t.Fatalf("search pattern = %q, want %q", m.searches[m.search].pattern, want)
	}
	if strings.Contains(m.searches[m.search].pattern, "+0.000000000") {
		t.Fatalf("search pattern contains rendered relative timestamp: %q", m.searches[m.search].pattern)
	}
}
