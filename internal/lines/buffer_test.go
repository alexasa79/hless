package lines

import (
	"os"
	"testing"
	"time"

	"github.com/alexasa79/hless/internal/utils"
)

func TestReadFileDoesNotHoldBufferLockWhileRefreshChannelIsFull(t *testing.T) {
	t.Parallel()

	r, w, err := os.Pipe()
	utils.Assert(err == nil, err)
	defer r.Close()
	defer w.Close()

	buf := NewBufferFromFileDescriptors(int(r.Fd()), 1)

	for i := 0; i < cap(buf.BufferRefreshChannel); i++ {
		w.Write([]byte("a"))
		<-buf.BufferRefreshChannel
	}

	for i := 0; i < cap(buf.BufferRefreshChannel); i++ {
		buf.BufferRefreshChannel <- BufferRefreshSignal(1)
	}

	w.Write([]byte("b"))

	deadline := time.After(time.Second)
	for {
		if buf.Size() == int64(cap(buf.BufferRefreshChannel)+1) {
			return
		}

		select {
		case <-deadline:
			t.Fatal("buffer lock remained blocked behind full refresh channel")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
