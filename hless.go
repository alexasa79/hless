package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"

	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/lines"
	"github.com/alexasa79/hless/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fred1268/go-clap/clap"
)

type config struct {
	Help       bool     `clap:"--help,-h"`
	Debug      string   `clap:"--debug"`
	Trace      string   `clap:"--trace"`
	Version    bool     `clap:"--version,-V"`
	InputFiles []string `clap:"trailing"`
	Profile    bool     `clap:"--profile"`
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
	  --profile         Enable CPU profiling.

  Supported modules with --debug and --trace:
    all, view, model, search, lines
`

var version string = "hless version 0.0.1"

var VL *hhlog.Logger
var SL *hhlog.Logger
var ML *hhlog.Logger
var LL *hhlog.Logger

func main() {
	cfg := &config{
		Help:       false,
		Debug:      "",
		Trace:      "",
		InputFiles: make([]string, 0),
	}
	if _, err := clap.Parse(os.Args[1:], cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if cfg.Profile {
		prof, err := os.Create("hless.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(prof)
		defer pprof.StopCPUProfile()
		defer prof.Close()
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
		hhlog.SetupLogging("hless.log")
		ML = hhlog.NewLogger("MODEL")
		VL = hhlog.NewLogger("VIEW")
		SL = hhlog.NewLogger("SEARCH")
		LL = hhlog.NewLogger("LINES")
		defer hhlog.CleanupLogging()

		setLevels := func(s string, newLevel hhlog.LogLevel) {
			for _, level := range strings.Split(s, ",") {
				switch level {
				case "model":
					ML.MinLevel = newLevel
				case "view":
					VL.MinLevel = newLevel
				case "search":
					SL.MinLevel = newLevel
				case "lines":
					LL.MinLevel = newLevel
				case "all":
					ML.MinLevel = newLevel
					VL.MinLevel = newLevel
					SL.MinLevel = newLevel
					LL.MinLevel = newLevel
				}
			}
		}

		setLevels(cfg.Debug, hhlog.DEBUG)
		setLevels(cfg.Trace, hhlog.TRACE)

		ML.Info("Logging at level %v", SL.MinLevel)
		VL.Info("Logging at level %v", VL.MinLevel)
		SL.Info("Logging at level %v", SL.MinLevel)
		LL.Info("Logging at level %v", LL.MinLevel)

		lines.LL = LL
		model.ML = ML
		model.SL = SL
		model.VL = VL
	}

	fi, err := in.Stat()
	if err != nil {
		fmt.Println("Error acquiring input file's attributes:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model.InitialModel(in, fi.Size()), tea.WithAltScreen())
	defer tea.ClearScreen()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error rendering the TUI: %v", err)
		os.Exit(1)
	}
}
