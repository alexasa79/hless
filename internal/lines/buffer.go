package lines

import (
	"os"
	"sync"
	"syscall"

	"github.com/alexasa79/hless/internal/utils"
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
	bytes    []byte
	streamFd int

	mx sync.Mutex

	BufferRefreshChannel chan BufferRefreshSignal
}

func NewBufferFromFile(filename string) *Buffer {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}

	fileSize := fi.Size()

	LL.Debug("Memory mapping %d bytes long file %s, file descriptor %d", fileSize, filename, f.Fd())

	var data []byte
	if fileSize > 0 {
		data, err = syscall.Mmap(int(f.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_SHARED)
		if err != nil {
			panic(err)
		}
	}

	return &Buffer{
		filename:             filename,
		fileSize:             fileSize,
		bytes:                data,
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
			buf.fileSize = int64(len(buf.bytes))
			buf.mx.Unlock()

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
	return b.bytes[i]
}

func (b *Buffer) Range(start, end int64) []byte {
	b.mx.Lock()
	defer b.mx.Unlock()

	return b.bytes[start:end]
}

func (b *Buffer) Refresh() bool {
	LL.Trace("Refreshing buffer")

	b.mx.Lock()
	defer b.mx.Unlock()

	switch b.bufferType {
	case bufferTypeFile:
		f, err := os.Open(b.filename)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			panic(err)
		}

		fileSize := fi.Size()

		if fileSize != b.fileSize {
			LL.Debug("File's size changed from %d to %d. Remapping", b.fileSize, fileSize)

			if len(b.bytes) > 0 {
				if err := syscall.Munmap(b.bytes); err != nil {
					panic(err)
				}
			}

			var data []byte
			if fileSize > 0 {
				data, err = syscall.Mmap(int(f.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_SHARED)
				if err != nil {
					panic(err)
				}
			}

			b.fileSize = fileSize
			b.bytes = data
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

func (b *Buffer) FastFind(
	from int64,
	to int64,
	what func(data []byte, offset int64) (int64, bool),
) (
	int64,
	bool,
) {
	b.mx.Lock()
	defer b.mx.Unlock()

	utils.Assert(to <= int64(len(b.bytes)), "to %d, buffer length %d", to, len(b.bytes))

	return what(b.bytes[from:to], from)
}
