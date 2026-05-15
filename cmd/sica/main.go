package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mojitrk/sica/config"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/server"
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

	logFile, err := os.OpenFile(filepath.Join(config.SicaDir(), "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

	deps := server.Deps{
		Store:       store,
		HabitsStore: habits.NewStore(store),
		TasksStore:  tasks.NewStore(store),
	}

	handler := server.New(deps)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		log.Printf("server starting on %s", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		os.Exit(0)
	}()

	p := tea.NewProgram(tui.New(deps), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("tui error: %v", err)
	}
}
