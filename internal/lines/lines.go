package lines

import (
	"bufio"
	"io"
	"math"

	"github.com/alexasa79/hless/internal/utils"
)

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
		input: make(chan string),
		lines: make([]string, 0),
		eof: false,
		fileReadProgress: make(chan float64),
	}

	go func() {
		var counter int64 = 0
		var lastProgress float64 = 0
        scanner := bufio.NewScanner(lines.file)
        for scanner.Scan() {
			s := scanner.Text()
			if lines.fileSize != math.MaxInt64 {
				counter += int64(len(s))
				t := float64(counter) / float64(lines.fileSize)
				if t - lastProgress > 0.001 {
					lastProgress = t
					lines.fileReadProgress <- t
				}
			}
            lines.input <- s
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

	var line string
	for utils.Len64(l.lines) < n {
		if interruptChannel != nil {
			select {
			case line, ok = <- l.input:
				if ok {
					l.lines = append(l.lines, line)
				} else {
					l.eof = true
					return
				}
			case prog, ok = <- l.fileReadProgress:
				return

			case interrupt, ok = <- interruptChannel:
				return
			}
		} else {
			if line, ok = <- l.input; ok {
				l.lines = append(l.lines, line)
			} else {
				l.eof = true
				break
			}
		}
	}

	return
}

