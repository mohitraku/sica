package main

import (
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mojitrk/sica/config"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/knowledge"
	"github.com/mojitrk/sica/internal/store/sqlite"
	"github.com/mojitrk/sica/internal/tasks"
	"github.com/mojitrk/sica/internal/tui"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logFile, err := os.OpenFile(filepath.Join(config.SicaDir(), "sica.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	store, err := sqlite.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()
	log.Printf("database opened: %s", cfg.Database.Path)

	hStore := habits.NewStore(store.DB)
	tStore := tasks.NewStore(store.DB)
	kStore := knowledge.NewStore(store.DB, cfg.Knowledge.Path)

	p := tea.NewProgram(tui.New(hStore, tStore, kStore), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}
