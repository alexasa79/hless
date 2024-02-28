package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fred1268/go-clap/clap"
)

type config struct {
	Help bool `clap:"--help,-h"`
	Debug  bool `clap:"--debug"`
	Version bool `clap:"--version,-V"`
	InputFiles []string `clap:"trailing"`
}

var help string = `Usage: hless [OPTIONS] FILE

  A utility for displaying and analyzing log files

Options:
  -h, --help
  -V, --version         Show the version and exit.
      --debug           Enable debug logging.
`

var version string = "hless version 0.0.1"

func main() {
	cfg := &config {
		Help: false,
		Debug: false,
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

	if cfg.Debug {
		f, err := tea.LogToFile("hless.log", "")
		if err != nil {
			fmt.Println("Error opening debug log:", err)
			os.Exit(1)
		}
		defer f.Close()
	} else {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("-----------------------------------------------------------")

    p := tea.NewProgram(initialModel(in), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error rendering the TUI: %v", err)
        os.Exit(1)
    }
}