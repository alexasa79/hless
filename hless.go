package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func parseArguments() (*string, error) {
	if (len(os.Args) != 2) {
		return nil, errors.New("missing input file")
	}
	return &os.Args[1], nil
}

func main() {
	file, err := parseArguments()
	if err != nil {
		fmt.Println("Error parsing arguments:", err)
		os.Exit(1)
	}
	in, err := os.Open(*file)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		os.Exit(1)
	}
	defer in.Close()

	f, err := tea.LogToFile("hless.log", "")
	if err != nil {
		fmt.Println("Error opening debug log:", err)
		os.Exit(1)
	}
	defer f.Close()

	log.Println("-----------------------------------------------------------")

    p := tea.NewProgram(initialModel(in), tea.WithAltScreen())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Error rendering the TUI: %v", err)
        os.Exit(1)
    }
}