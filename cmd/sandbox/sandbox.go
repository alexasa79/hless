package main

import (
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/tui"
)

var SL = hhlog.NewLogger("SB")

func main() {
	hhlog.SetupLogging("sandbox.log")
	defer hhlog.CleanupLogging()

	SL.MinLevel = hhlog.TRACE
	tui.TL.MinLevel = hhlog.TRACE

	ui := tui.NewTUI()
	ui.SpawnMainLoop()

	var s string

	redStyle := tui.NewStyle()
	redStyle.Foreground = tui.NewColor(0xff, 0, 0)
	// boldStyle := tui.Style{Bold: tui.PropertyTrue}

loop:
	for {
		input := ui.WaitForInput()
		SL.Debug("Received input: %s", input.Code)

		switch {
		case input.Code == "ctrl+c":
			break loop
		case input.Code == "q":
			break loop
		default:
			s += input.Code
			ui.Write(0, 0, s)
			ui.Write(1, 0, s)
			ui.ApplyStyle(1, 0, len(s), &redStyle)
			ui.Write(2, 0, s)
			// ui.ApplyStyle(2, 0, len(s), &boldStyle)
			ui.Write(3, 0, s)
			ui.Write(4, 0, s)
			ui.Write(5, 0, s)
			ui.Write(6, 0, s)
		}

		ui.Render()
	}

	SL.Info("Event loop finished. Exiting")

	ui.Exit()
}
