package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/fred1268/go-clap/clap"
)

type config struct {
	Help bool `clap:"--help,-h"`
	Lines int `clap:"--size,-s"`
	Filename string `clap:"trailing"`
}

var help string = `Usage: log-gen [OPTS] FILENAME

  A utility for generating dummy log files.

Options:
  -h, --help
  -s, --size [LINES]   Specifies the number of lines in the file.
`

func main() {
	cfg := &config{
		Lines: 100000,
	}
	if _, err := clap.Parse(os.Args[1:], cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if cfg.Help || cfg.Filename == "" {
		fmt.Println(help)
		os.Exit(0)
	}

	hhlog.SetupLogging(cfg.Filename)
	defer hhlog.CleanupLogging()

	L := hhlog.NewLogger("LOG")

	s := `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	for i := 0; i < cfg.Lines; i++ {
		n := rand.Intn(len(s))
		L.Info("L:%-10d %s", i, s[:n])
	}
}