package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/alexasa79/hless/internal/hhlog"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sys/unix"
)

type Model struct {
	bytes string
	mutex sync.Mutex
	ch    chan int
}

var LL *hhlog.Logger

func InitialModel() *Model {
	return &Model{
		ch: make(chan int, 1000),
	}
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		sig, ok := <-m.ch
		if ok {
			return sig
		} else {
			return nil
		}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	LL.Trace("Asked to update")

	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "q" {
			cmds = append(cmds, tea.Quit)
		}
	case int:
		n := int(msg)
		LL.Trace("Received %d bytes", n)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	LL.Trace("Asked to render")

	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.bytes
}

func main() {
	hhlog.SetupLogging("stdin.log")
	defer hhlog.CleanupLogging()

	LL = hhlog.NewLogger("STDIN")
	LL.MinLevel = hhlog.TRACE

	m := InitialModel()

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

	go func() {
		buffer := make([]byte, 1024)

		for {
			n, err := unix.Read(savedStdin, buffer)
			if err != nil {
				if err == unix.EAGAIN {
					continue
				} else {
					break
				}
			}

			if n > 0 {
				m.mutex.Lock()
				m.bytes += string(buffer[:n])
				m.ch <- n
				m.mutex.Unlock()
			} else if n == 0 {
				break
			}
		}
	}()

	// buf := make([]byte, 1024)
	// for {
	// 	n, _ := newStdin.Read(buf)
	// 	if n > 0 {
	// 		m.mutex.Lock()
	// 		fmt.Print(m.bytes)
	// 		m.mutex.Unlock()
	// 		fmt.Print(string(buf[:n]))
	// 	}
	// }

	p := tea.NewProgram(m, tea.WithInput(newStdin))
	// defer tea.ExitAltScreen()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error rendering the TUI: %v", err)
		os.Exit(1)
	}
}
