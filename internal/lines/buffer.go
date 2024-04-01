package lines

import (
	"os"

	"golang.org/x/exp/mmap"
)

type bufferType int

const (
	bufferTypeFile bufferType = iota
	bufferTypeString
)

type Buffer struct {
	fileSize int64
	m        *mmap.ReaderAt

	bufferType bufferType
	bytes      []byte
}

func NewBufferFromFile(filename string) *Buffer {
	stat, err := os.Stat(filename)
	if err != nil {
		panic(err)
	}
	fileSize := stat.Size()

	m, err := mmap.Open(filename)
	if err != nil {
		panic(err)
	}

	return &Buffer{
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
	}
	panic("unreachable")
}
