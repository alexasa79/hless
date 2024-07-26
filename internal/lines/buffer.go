package lines

import (
	"os"
	"syscall"
	"time"

	"github.com/alexasa79/hless/internal/utils"
	"golang.org/x/exp/mmap"
)

type bufferType int

const (
	bufferTypeFile bufferType = iota
	bufferTypeString
	bufferTypeStream
)

type Buffer struct {
	bufferType bufferType

	filename string
	fileSize int64
	m        *mmap.ReaderAt
	bytes    []byte
	streamFd int
}

func NewBufferFromFile(filename string) *Buffer {
	stat, err := os.Stat(filename)
	if err != nil {
		panic(err)
	}
	fileSize := stat.Size()

	LL.Debug("Memory mapping %d bytes long file %s", fileSize, filename)

	m, err := mmap.Open(filename)
	if err != nil {
		panic(err)
	}

	return &Buffer{
		filename:   filename,
		fileSize:   fileSize,
		m:          m,
		bufferType: bufferTypeFile,
	}
}

func NewBufferFromString(s string) *Buffer {
	return &Buffer{
		fileSize:   int64(len(s)),
		bufferType: bufferTypeString,
		bytes:      []byte(s),
	}
}

func readSome(fd int) []byte {
	b := make([]byte, 65536)

	n, err := syscall.Read(fd, b)
	if err != nil {
		LL.Warning("%s", err)
	} else {
		b = b[:n]
	}

	LL.Trace("Read %d bytes from the stream", len(b))

	return b
}

func NewBufferFromStdin() *Buffer {
	// Create a duplicate of stdin file and file descriptor
	nfd, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	os.Stdin.Close()

	return &Buffer{
		fileSize:   int64(0),
		bufferType: bufferTypeStream,
		bytes:      make([]byte, 0),
		streamFd:   nfd,
	}
}

func (b *Buffer) Close() {
	b.m.Close()
}

func (b *Buffer) Size() int64 {
	return b.fileSize
}

func (b *Buffer) At(i int64) byte {
	switch b.bufferType {
	case bufferTypeFile:
		return b.m.At(int(i))
	case bufferTypeString:
		return b.bytes[i]
	case bufferTypeStream:
		return b.bytes[i]
	}
	panic("unreachable")
}

func (b *Buffer) Range(start, end int64) []byte {
	switch b.bufferType {
	case bufferTypeFile:
		buf := make([]byte, end-start)
		b.m.ReadAt(buf, start)
		return buf
	case bufferTypeString:
		return b.bytes[start:end]
	case bufferTypeStream:
		return b.bytes[start:end]
	}
	panic("unreachable")
}

func (b *Buffer) Refresh() bool {
	LL.Trace("Refreshing buffer")

	switch b.bufferType {
	case bufferTypeFile:
		stat, err := os.Stat(b.filename)
		if err != nil {
			return false
		}
		fileSize := stat.Size()
		if fileSize != b.fileSize {
			LL.Debug("File's size changed from %d to %d. Remapping", b.fileSize, fileSize)
			m, err := mmap.Open(b.filename)

			utils.Assert(err == nil, err)

			b.fileSize = fileSize
			b.m.Close()
			b.m = m
			return true
		} else {
			time.Sleep(100 * time.Millisecond)
		}
		return false
	case bufferTypeString:
		return false
	case bufferTypeStream:
		res := false
		buf := readSome(b.streamFd)
		if len(buf) > 0 {
			LL.Debug("Read another %d bytes. Got %d bytes total", len(buf), len(b.bytes)+len(buf))
			res = true
		}
		b.bytes = append(b.bytes, buf...)
		b.fileSize = int64(len(b.bytes))
		return res
	}
	return false
}
