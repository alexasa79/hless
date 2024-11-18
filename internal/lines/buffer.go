package lines

import (
	"os"
	"sync"
	"syscall"

	"github.com/alexasa79/hless/internal/utils"
	"golang.org/x/exp/mmap"
	"golang.org/x/sys/unix"
)

type bufferType int
type BufferRefreshSignal int

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

	mx sync.Mutex

	BufferRefreshChannel chan BufferRefreshSignal
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
		filename:             filename,
		fileSize:             fileSize,
		m:                    m,
		bufferType:           bufferTypeFile,
		BufferRefreshChannel: make(chan BufferRefreshSignal, 10000),
	}
}

func NewBufferFromString(s string) *Buffer {
	return &Buffer{
		fileSize:             int64(len(s)),
		bufferType:           bufferTypeString,
		bytes:                []byte(s),
		BufferRefreshChannel: make(chan BufferRefreshSignal, 10000),
	}
}

func readFile(buf *Buffer, fd int, bufferSize int) {
	buffer := make([]byte, bufferSize)

	for {
		n, err := unix.Read(fd, buffer)
		if err != nil {
			LL.Trace("Error %v reading from the buffer", err)
			if err == unix.EAGAIN {
				continue
			} else {
				break
			}
		}

		if n > 0 {
			LL.Trace("Read %d bytes from the input", n)

			buf.mx.Lock()
			buf.bytes = append(buf.bytes, buffer[:n]...)
			buf.mx.Unlock()

			buf.fileSize = int64(len(buf.bytes))

			buf.BufferRefreshChannel <- BufferRefreshSignal(n)
		} else if n == 0 {
			break
		}
	}
}

func NewBufferFromFileDescriptors(fd int, bufferSize int) *Buffer {
	buf := &Buffer{
		fileSize:             int64(0),
		bufferType:           bufferTypeStream,
		bytes:                make([]byte, 0),
		streamFd:             fd,
		BufferRefreshChannel: make(chan BufferRefreshSignal, 10000),
	}
	go readFile(buf, fd, bufferSize)
	return buf
}

func NewBufferFromStdin() (*Buffer, *os.File) {
	savedStdin, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	err = syscall.Close(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	newStdin, err := os.Open("/dev/tty")
	if err != nil {
		panic(err)
	}

	buf := &Buffer{
		fileSize:             int64(0),
		bufferType:           bufferTypeStream,
		bytes:                make([]byte, 0),
		streamFd:             savedStdin,
		BufferRefreshChannel: make(chan BufferRefreshSignal, 10000),
	}

	go readFile(buf, savedStdin, 65536)

	return buf, newStdin
}

func (b *Buffer) Close() {
	b.m.Close()
}

func (b *Buffer) Size() int64 {
	b.mx.Lock()
	defer b.mx.Unlock()

	return b.fileSize
}

func (b *Buffer) Name() string {
	switch b.bufferType {
	case bufferTypeFile:
		return b.filename
	case bufferTypeString:
		return "string"
	case bufferTypeStream:
		return "standard input"
	}
	panic("unreachable")
}

func (b *Buffer) At(i int64) byte {
	b.mx.Lock()
	defer b.mx.Unlock()

	utils.Assert(i >= 0, i)

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
	b.mx.Lock()
	defer b.mx.Unlock()

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

	b.mx.Lock()
	defer b.mx.Unlock()

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
			LL.Trace("New buffer size %d", b.fileSize)
			return true
		}
		return false
	case bufferTypeString:
		return false
	case bufferTypeStream:
		return false
	}
	return false
}
