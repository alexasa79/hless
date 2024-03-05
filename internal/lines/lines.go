package lines

import (
	"bufio"
	"io"
	"math"

	hlesslog "github.com/alexasa79/hless/internal/logging"
	"github.com/alexasa79/hless/internal/utils"
)

var LL *hlesslog.Logger

type Lines struct {
	file io.Reader
	fileSize int64
	input chan string
	lines []string
	eof bool
	fileReadProgress chan float64
}

func NewLines(file io.Reader, fileSize int64) *Lines {
	lines := &Lines{
		file: file,
		fileSize: fileSize,
		input: make(chan string, 10000),
		lines: make([]string, 0),
		eof: false,
		fileReadProgress: make(chan float64, 10000),
	}

	go func() {
		LL.Info("Logging routine started")

		var counter int64 = 0
		var lastProgress float64 = 0
        scanner := bufio.NewScanner(lines.file)
        for scanner.Scan() {
			s := scanner.Text()
			LL.Trace("Read %d byte line from input", len(s))
			if lines.fileSize != math.MaxInt64 {
				counter += int64(len(s))
				t := float64(counter) / float64(lines.fileSize)
				if t - lastProgress > 0.001 {
					lastProgress = t
					LL.Trace("Reporting progress %v", t)
					lines.fileReadProgress <- t
				}
			}
            lines.input <- s
			LL.Trace("Sent %d byte line to the channel", len(s))
        }
		lines.fileReadProgress <- 1.0
        close(lines.input) // Close the channel once done reading
		close(lines.fileReadProgress)
    }()

	return lines
}

func (l *Lines) Nth(n int64) string {
	return l.lines[n]
}

func (l *Lines) Nlines() int64 {
	return int64(len(l.lines))
}

func (l *Lines) Eof() bool {
	return l.eof
}

func (l *Lines) FileSize() int64 {
	return l.fileSize
}

func (l *Lines) Populate(n int64, interruptChannel chan bool) (prog float64, interrupt bool) {
	if l.eof {
		return 0, false
	}

	prog = 0
	interrupt = false
	ok := true

	LL.Debug("Asked to populate to line %d", n)

	var line string
	for utils.Len64(l.lines) < n {
		LL.Trace("Reading line after %d lines", len(l.lines))
		if interruptChannel != nil {
			select {
			case line, ok = <- l.input:
				if ok {
					LL.Trace("Received %d byte line", len(line))
					l.lines = append(l.lines, line)
				} else {
					LL.Trace("Encountered what looks like an error")
					l.eof = true
					return
				}
			case prog, ok = <- l.fileReadProgress:
				LL.Trace("Received progress report %v", prog)
				return

			case interrupt, ok = <- interruptChannel:
				LL.Trace("Received interrupt signal")
				return
			}
		} else {
			if line, ok = <- l.input; ok {
				LL.Trace("Received %d byte line", len(line))
				l.lines = append(l.lines, line)
			} else {
				LL.Trace("Encountered what looks like an error")
				l.eof = true
				break
			}
		}
	}

	return
}

