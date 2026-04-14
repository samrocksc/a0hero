package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/samrocksc/a0hero/client"
	"github.com/samrocksc/a0hero/logger"
	clientmod "github.com/samrocksc/a0hero/modules/clients"
	"github.com/samrocksc/a0hero/tui"
)

func main() {
	// Setup logging
	logDir := "/Users/sam/GitHub/work/a0hero/logs"
	os.MkdirAll(logDir, 0755)
	if err := logger.Setup(true, logDir); err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	logger.Info("TUI integration test starting")

	// Setup config
	configDir := "/Users/sam/.config/a0hero"
	os.MkdirAll(configDir, 0755)

	cfg := &client.Config{
		Name:         "acul-tryout",
		Domain:       "acul-tryout.cic-demo-platform.auth0app.com",
		ClientID:     "tdxxRErnhXjAJdW90nzDVceA5oFH7HZx",
		ClientSecret: "dCYiiSZKWXZ7R7zXSDRATYNoo6KtrB0-Ynf2SDpl6G72uourXXyol-uo355Td8it",
	}
	configPath := filepath.Join(configDir, "acul-tryout.yaml")
	if err := cfg.WriteFile(configPath); err != nil {
		log.Fatal(err)
	}

	// Create API client
	api, err := client.NewClientFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create client", "error", err)
		log.Fatal(err)
	}

	// Verify we can fetch clients
	ctx := context.Background()
	clients, err := clientmod.New(api).List(ctx)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("got clients", "count", len(clients))

	// Create app with the connected API
	app := tui.NewApp(configDir, true)
	app.SetAPI(api, cfg)
	
	logger.Info("App created with API, starting program...")

	// Create a channel to receive the final model
	done := make(chan struct{})

	go func() {
		p := tea.NewProgram(app, tea.WithAltScreen())
		
		// Wait a moment for init
		time.Sleep(500 * time.Millisecond)
		
		// Send WindowSizeMsg
		p.Send(tea.WindowSizeMsg{Width: 120, Height: 40})
		
		// Wait for initial load
		time.Sleep(2 * time.Second)
		
		// Check if loaded
		if app.IsConnected() {
			logger.Info("app is connected")
		} else {
			logger.Error("app is NOT connected")
		}
		
		// Try sending 'l' key to switch to Clients
		p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		time.Sleep(1 * time.Second)
		
		// Try sending 'l' again
		p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		time.Sleep(2 * time.Second)
		
		// Check for edit overlay
		e := app.GetEditOverlay()
		logger.Info("edit overlay state", "nil", e == nil)
		
		// Try sending 'e' key
		p.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		time.Sleep(3 * time.Second)
		
		// Check for edit overlay again
		e = app.GetEditOverlay()
		logger.Info("edit overlay after 'e'", "nil", e == nil)
		
		p.Quit()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("program finished")
	case <-time.After(15 * time.Second):
		logger.Error("timeout - program took too long")
	}

	logger.Info("test completed")
}
