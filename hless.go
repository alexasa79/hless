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

	_ "net/http/pprof"
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
    all, view, model, search, lines, io
`

var gitrev string
var buildTime string

var version string = "hless version 0.0.1"

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
	}

	if cfg.Help {
		fmt.Println(help)
		os.Exit(0)
	}

	if cfg.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	if cfg.Debug != "" || cfg.Trace != "" {
		hhlog.SetupLogging("hless.log")
		defer hhlog.CleanupLogging()

		setLevels := func(s string, newLevel hhlog.LogLevel) {
			for _, level := range strings.Split(s, ",") {
				switch level {
				case "model":
					model.ML.MinLevel = newLevel
				case "view":
					model.VL.MinLevel = newLevel
				case "search":
					model.SL.MinLevel = newLevel
				case "lines":
					lines.LL.MinLevel = newLevel
				case "io":
					lines.LIO.MinLevel = newLevel
				case "all":
					model.ML.MinLevel = newLevel
					model.VL.MinLevel = newLevel
					model.SL.MinLevel = newLevel
					lines.LL.MinLevel = newLevel
					lines.LIO.MinLevel = newLevel
				}
			}
		}

		setLevels(cfg.Debug, hhlog.DEBUG)
		setLevels(cfg.Trace, hhlog.TRACE)

		model.ML.Info("Logging at level %v", model.ML.MinLevel)
		model.VL.Info("Logging at level %v", model.VL.MinLevel)
		model.SL.Info("Logging at level %v", model.SL.MinLevel)
		lines.LL.Info("Logging at level %v", lines.LL.MinLevel)
		lines.LIO.Info("Logging at level %v", lines.LIO.MinLevel)
	}

	var buffer *lines.Buffer
	var stdin *os.File = os.Stdin

	if len(cfg.InputFiles) != 1 {
		buffer, stdin = lines.NewBufferFromStdin()
	} else {
		in, err := os.Open(cfg.InputFiles[0])
		if err != nil {
			fmt.Println("Error opening input file:", err)
			os.Exit(1)
		}
		defer in.Close()

		buffer = lines.NewBufferFromFile(in.Name())
		defer buffer.Close()
	}

	p := tea.NewProgram(model.InitialModel(buffer, gitrev, buildTime), tea.WithAltScreen(), tea.WithInput(stdin))
	defer tea.ExitAltScreen()
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error rendering the TUI: %v", err)
		os.Exit(1)
	}
}
