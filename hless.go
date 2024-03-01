package main

import (
	"fmt"
	"os"

	"github.com/alexasa79/hless/internal/ll"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fred1268/go-clap/clap"
)

type config struct {
	Help bool `clap:"--help,-h"`
	Debug  string `clap:"--debug"`
	Trace string `clap:"--trace"`
	Version bool `clap:"--version,-V"`
	InputFiles []string `clap:"trailing"`
}

var help string = `Usage: hh [OPTIONS] FILE

  A utility for displaying and analyzing log files

Options:
  -h, --help
  -V, --version         Show the version and exit.
      --debug [MODS]    Enable debug logging for specified comma separated
	                    list of modules.
      --trace [MODS]    Enable trace logging for specified comma separated
	                    list of modules. Implies --debug.

  Supported modules with --debug and --trace:
    all, view, model, search
`

var version string = "hless version 0.0.1"

var ViewLogger *ll.Logger
var SearchLogger *ll.Logger
var ModelLogger *ll.Logger

func main() {
	cfg := &config {
		Help: false,
		Debug: "",
		Trace: "",
		InputFiles: make([]string, 0),
	}
	if _, err := clap.Parse(os.Args[1:], cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if cfg.Help {
		fmt.Println(help)
		os.Exit(0)
	}

	if cfg.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if len(cfg.InputFiles) != 1 {
		fmt.Fprintln(os.Stderr, "Expected single input file")
		os.Exit(1)
	}

	in, err := os.Open(cfg.InputFiles[0])
	if err != nil {
		fmt.Println("Error opening input file:", err)
		os.Exit(1)
	}
	defer in.Close()

	if cfg.Debug != "" || cfg.Trace != "" {
		ll.SetupLogging("hless.log")
		ViewLogger = ll.NewLogger("VIEW  ")
		SearchLogger = ll.NewLogger("SEARCH")
		ModelLogger = ll.NewLogger("MODEL ")
		defer ll.CleanupLogging()
	}

	fi, err := in.Stat()
	if err != nil {
		fmt.Println("Error acquiring input file's attributes:", err)
		os.Exit(1)
	}

    p := tea.NewProgram(initialModel(in, fi.Size()), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error rendering the TUI: %v", err)
        os.Exit(1)
    }
}