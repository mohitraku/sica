package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/mojitrk/sica/internal/storage"
	"github.com/mojitrk/sica/internal/tui"
)

func main() {
	dataDir := sicaDir()

	db, err := storage.Open(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	hStore := storage.NewHabitStore(db)
	eStore := storage.NewEntryStore(db)

	model := tui.New(hStore, eStore)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func sicaDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".sica"
	}
	return home + "/.sica"
}
