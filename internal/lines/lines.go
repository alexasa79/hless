package lines

import (
	"fmt"
	"io"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
)

var LL *hhlog.Logger

func LineComp(a, b *Line) int {
	if a.Offset < b.Offset {
		return -1
	} else if a.Offset == b.Offset {
		return 0
	} else {
		return 1
	}
}

type Line struct {
	Offset int64
	Text string
	Name string
	rawLength int64
}

type Lines struct {
	file io.ReadSeeker
	fileSize int64
	eof bool

	buf []byte
	offset int64
	section string
	sectionIndex int64

	lt *avltree.AVLTree[* Line]
}

func NewLines(file io.ReadSeeker, fileSize int64) *Lines {
	lines := &Lines{
		file: file,
		fileSize: fileSize,
		eof: false,

		buf: make([]byte, 0),
		offset: 0,
		section: "",
		sectionIndex: 0,

		lt: avltree.NewAVLTree[* Line](LineComp),
	}
	return lines
}

func (l *Lines) Eof() bool {
	return l.eof
}

func (l *Lines) FileSize() int64 {
	return l.fileSize
}

func (l *Lines) readForward() {
	if l.eof {
		return
	}

	buf := make([]byte, 65536)
	n, err := l.file.Read(buf)
	if err != nil {
		l.eof = true
	} else {
		start := int64(0)
		i := int64(0)
		buf = buf[:n]
		buf = append(l.buf, buf...)
		for {
			if i >= utils.Len64(buf) {
				break
			}

			if buf[i] == '\n' {
				text := string(buf[start:i])
				if text[len(text) - 1] == '\r' {
					text = text[:len(text) - 1]
				}

				name := ""
				if l.section == "" {
					name = fmt.Sprintf("%d", l.sectionIndex + 1)
				} else {
					name = fmt.Sprintf("%s%d", l.section, l.sectionIndex + 1)
				}

				line := Line {
					Offset: l.offset + start,
					Text: text,
					Name: name,
					rawLength: i - start + 1,
				}

				l.lt.Insert(&line)

				start = i + 1
				l.sectionIndex++
			}
			i++
		}
		l.buf = buf[i:]
	}
	l.offset += int64(n)
}

func (l *Lines) First() *Line {
	l.readForward()
	t := Line{ Offset: 0 }
	return *l.lt.Get(&t)
}

func (l *Line) Last() *Line {
	return nil
}

func (l *Lines) next(line *Line, iterations int) (next *Line) {
	if iterations > 3 {
		panic("Too many iterations trying to get next line")
	}

	next = *l.lt.Successor(line)
	if next == nil && l.eof {
		return nil
	}
	if next == nil {
		l.readForward()
		return l.next(line, iterations + 1)
	} else {
		if line.Offset + int64(line.rawLength) == int64(next.Offset) {
			return next
		} else {
			l.readForward()
			return l.next(line, iterations + 1)
		}
	}
}

func (l *Lines) Next(line *Line) (next *Line) {
	return l.next(line, 0)
}

func (l *Lines) Prev(line *Line) (prev *Line) {
	prev = *l.lt.Predecessor(line)
	return
}