package tui

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"unicode"
	"unsafe"

	"github.com/alexasa79/hless/internal/hhlog"
)

var TL = hhlog.NewLogger("TUI")

type Input struct {
	Code   string
	Shift  bool
	Signal bool
}

type Winsize struct {
	Rows    uint16
	Cols    uint16
	Xpixels uint16
	Ypixels uint16
}

type TUI struct {
	rows        []string
	styles      [][]*Style
	lock        sync.Mutex
	exitChannel chan struct{}
	inputCh     chan Input
	wg          sync.WaitGroup
	nRows       int
	nCols       int
}

func NewTUI() (tui *TUI) {
	return &TUI{
		exitChannel: make(chan struct{}),
		inputCh:     make(chan Input, 1),
	}
}

func clearScreen() {
	fmt.Printf("\033[2J\033[H")
}

func hideCursor() {
	fmt.Printf("\033[?25l")
}

func showCursor() {
	fmt.Printf("\033[?25h")
}

func moveCursor(row int, col int) {
	fmt.Printf("\033[%d;%dH", row, col)
}

func setEcho(enable bool) {
	var termios syscall.Termios
	fd := os.Stdin.Fd()
	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGETA), uintptr(unsafe.Pointer(&termios)), 0, 0, 0); errno != 0 {
		fmt.Printf("Error getting termios: %v\n", errno)
		return
	}

	if enable {
		termios.Lflag |= syscall.ICANON | syscall.ECHO
	} else {
		termios.Lflag &^= syscall.ICANON | syscall.ECHO
	}

	if _, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSETA), uintptr(unsafe.Pointer(&termios)), 0, 0, 0); errno != 0 {
		fmt.Printf("Error setting termios: %v\n", errno)
	}
}

func (tui *TUI) getTerminalSize() {
	ws := Winsize{}
	retCode, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)

	if int(retCode) == -1 {
		panic(errno)
	}

	tui.lock.Lock()
	tui.nRows = int(ws.Rows)
	tui.nCols = int(ws.Cols)
	TL.Info("Terminal size set to %dx%d", tui.nRows, tui.nCols)
	tui.lock.Unlock()
}

func (tui *TUI) redimension() {
	if len(tui.rows) < tui.nCols {
		emptyStyles := make([]*Style, tui.nCols)
		for len(tui.rows) < tui.nRows {
			tui.rows = append(tui.rows, "")
			tui.styles = append(tui.styles, emptyStyles)
		}
	}

	if len(tui.rows[0]) < tui.nCols {
		emptyStyles := make([]*Style, tui.nCols)
		for i := 0; i < len(tui.rows); i++ {
			tui.rows[i] += fmt.Sprintf("%-*s", tui.nCols-len(tui.rows[i]), " ")
			tui.styles[i] = append(tui.styles[i], emptyStyles[:tui.nCols-len(tui.styles[i])]...)
		}
	}
}

func (tui *TUI) Write(row int, col int, what string) {
	tui.lock.Lock()
	defer tui.lock.Unlock()

	TL.Trace("Write at %dx%d: \"%s\"", row, col, what)

	end := min(len(what)+col, tui.nCols)
	tui.redimension()
	tui.rows[row] = tui.rows[row][:col] + what + tui.rows[row][end:]

	TL.Debug("Set row %d to: \"%s\"", row, strings.TrimRightFunc(tui.rows[row], unicode.IsSpace))
}

func (tui *TUI) ApplyStyle(row int, colStart int, colEnd int, style *Style) {
	tui.lock.Lock()
	defer tui.lock.Unlock()

	clone := new(Style)
	*clone = *style
	for i := colStart; i < min(colEnd+1, tui.nCols); i++ {
		tui.styles[row][i] = clone
	}
}

func (tui *TUI) Render() {
	tui.lock.Lock()
	defer tui.lock.Unlock()

	clearScreen()

	for i := 0; i < tui.nRows; i++ {
		var style *Style
		ins := strings.TrimRightFunc(tui.rows[i], unicode.IsSpace)
		s := ""
		for j := 0; j < len(ins); j++ {
			if style != tui.styles[i][j] {
				style = tui.styles[i][j]
				s += applyStyle(style)
			}
			s += string(ins[j])
		}
		if style != nil {
			applyStyle(nil)
		}
		fmt.Printf(s + "\n")
	}
}

func (tui *TUI) WaitForInput() Input {
	TL.Debug("Waiting for input")
	return <-tui.inputCh
}

func bytesToInt(b []byte) int {
	bb := make([]byte, 4)
	copy(bb, b)
	return int(binary.LittleEndian.Uint32(bb))
}

func (tui *TUI) bytesToInput(code int) {
	// Useful link: https://github.com/charmbracelet/bubbletea/blob/master/key.go
	switch {
	case code == 0x445b1b: // esc [ D
		tui.inputCh <- Input{Code: "left"}
	case code == 0x415b1b: // esc [ A
		tui.inputCh <- Input{Code: "up"}
	case code == 0x435b1b: // esc [ C
		tui.inputCh <- Input{Code: "right"}
	case code == 0x425b1b: // esc [ B
		tui.inputCh <- Input{Code: "down"}
	case code == int(' '):
		tui.inputCh <- Input{Code: " "}
	case code >= int('a') && code <= int('z'):
		tui.inputCh <- Input{Code: string(rune(code))}
	case code >= int('A') && code <= int('Z'):
		tui.inputCh <- Input{Code: string(rune(code)), Shift: true}
	default:
		TL.Info("Received unsupported key code %#x", code)
	}
}

func (tui *TUI) SpawnMainLoop() {
	tui.wg.Add(1)
	go tui.mainLoop()
}

func (tui *TUI) mainLoop() {
	defer tui.wg.Done()

	sigwinchChannel := make(chan os.Signal, 1)
	sigintChannel := make(chan os.Signal, 1)

	signal.Notify(sigwinchChannel, syscall.SIGWINCH)
	signal.Notify(sigintChannel, syscall.SIGINT)

	clearScreen()

	hideCursor()
	defer showCursor()

	setEcho(false)
	defer setEcho(true)

	tui.getTerminalSize()

	go tui.inputReader()

loop:
	for {
		select {
		case <-sigwinchChannel:
			TL.Debug("Received SIGWINCH")
			tui.getTerminalSize()

		case <-sigintChannel:
			TL.Debug("Received SIGINT")
			tui.inputCh <- Input{Code: "ctrl+c"}
			break loop

		case <-tui.exitChannel:
			TL.Debug("Main loop received exit notification")
			break loop
		}
	}

	TL.Info("MainLoop finished")
}

func (tui *TUI) Exit() {
	close(tui.exitChannel)
	tui.wg.Wait()
}

func (tui *TUI) inputReader() {
	zeros := make([]byte, 4)
	bytes := make([]byte, 4)

	for {
		copy(bytes, zeros)

		n, err := os.Stdin.Read(bytes)
		if err != nil {
			panic(err)
		}

		var s string
		for i := 0; i < n; i++ {
			s += fmt.Sprintf("%d ", bytes[i])
		}

		TL.Info("Received input bytes %s, %#x", strings.TrimSpace(s), bytesToInt(bytes))
		tui.bytesToInput(bytesToInt(bytes))
	}
}

func (tui *TUI) Rows() int {
	tui.lock.Lock()
	defer tui.lock.Unlock()

	return tui.nRows
}

func (tui *TUI) Columns() int {
	tui.lock.Lock()
	defer tui.lock.Unlock()

	return tui.nCols
}
