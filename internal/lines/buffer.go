package lines

import (
	"os"
	"syscall"

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
	b := make([]byte, 0)

	for {
		temp := make([]byte, 8192)
		n, err := syscall.Read(fd, temp)

		if err == syscall.EAGAIN || n == 0 {
			break
		}

		b = append(b, temp[:n]...)
	}

	return b
}

func NewBufferFromStdin() *Buffer {
	// Create a duplicate of stdin file and file descriptor
	nfd, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	os.Stdin.Close()

	syscall.SetNonblock(nfd, true)

	b := readSome(nfd)

	LL.Debug("Read %d bytes from the stream", len(b))

	return &Buffer{
		fileSize:   int64(len(b)),
		bufferType: bufferTypeStream,
		bytes:      b,
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

func (b *Buffer) Refresh() {
	switch b.bufferType {
	case bufferTypeFile:
		stat, err := os.Stat(b.filename)
		if err != nil {
			return
		}
		fileSize := stat.Size()
		if fileSize != b.fileSize {
			LL.Debug("File's size changed from %d to %d. Remapping", b.fileSize, fileSize)
			m, err := mmap.Open(b.filename)

			utils.Assert(err == nil, err)

			b.fileSize = fileSize
			b.m.Close()
			b.m = m
		}
	case bufferTypeString:
		return
	case bufferTypeStream:
		return
	}
}