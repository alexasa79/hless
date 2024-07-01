package model

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

type History struct {
	Version int `json:"version"`
	History []string `json:"history"`
	temp string

	cursor int
}

func NewHistory() *History {
	return &History{
		History: make([]string, 0),
	}
}

func historyFilename() string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	homeDir := usr.HomeDir
	cacheDir := filepath.Join(homeDir, ".cache")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		// Create the cache directory
		err := os.Mkdir(cacheDir, 0755)
		if err != nil {
			ML.Info("Failed to create ~/.cache. %v", err)
			return ""
		}
	}

	return filepath.Join(cacheDir, "hless.history")
}

func (h *History) Load() {
	filename := historyFilename()
	if filename == "" {
		return
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		ML.Info("History file could not be found. Proceeding without...")
		return
	}

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		ML.Info("Failed to open history file: %v. Proceeding without...", err)
		return
	}
	defer file.Close()

    decoder := json.NewDecoder(file)
    if err := decoder.Decode(h); err != nil {
        panic(err)
    }

	h.cursor = len(h.History)
}

func (h *History) Save() {
	filename := historyFilename()

	// Open the file for appending or create if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		ML.Info("Failed to open history file: %v", err)
		return
	}
	defer file.Close()

	h.Version = 1

    encoder := json.NewEncoder(file)
    if err := encoder.Encode(h); err != nil {
        panic(err)
    }
}

func (h *History) Append(s string) {
	if h.History[len(h.History)-1] != s {
		// Do not append same string twice
		h.History = append(h.History, s)
	}
	h.cursor = len(h.History)
}

func (h *History) Prev(current string) string {
	if h.cursor == len(h.History) {
		h.temp = current
	}

	if h.cursor == 0 {
		return current
	}

	h.cursor--

	return h.History[h.cursor]
}

func (h *History) Next(current string) string {
	if h.cursor == len(h.History) - 1 {
		h.cursor++
		s := h.temp
		h.temp = ""
		return s
	}

	if h.cursor == len(h.History) {
		return current
	}

	h.cursor++

	return h.History[h.cursor]
}

func (h *History) ResetCursor() {
	h.cursor = len(h.History)
	h.temp = ""
}
