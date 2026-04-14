package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/samrocksc/a0hero/client"
	"github.com/samrocksc/a0hero/logger"
	clientmod "github.com/samrocksc/a0hero/modules/clients"
	"github.com/samrocksc/a0hero/tui/views"
)

func main() {
	// Setup logging
	logDir := "/Users/sam/GitHub/work/a0hero/logs"
	os.MkdirAll(logDir, 0755)
	if err := logger.Setup(true, logDir); err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	logger.Info("test starting")

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

	logger.Info("config written", "path", configPath)

	// Create API client
	api, err := client.NewClientFromConfig(cfg)
	if err != nil {
		logger.Error("failed to create client", "error", err)
		log.Fatal(err)
	}
	logger.Info("API client created successfully")

	// Test fetching clients directly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clients, err := clientmod.New(api).List(ctx)
	if err != nil {
		logger.Error("failed to list clients", "error", err)
		log.Fatal(err)
	}
	logger.Info("fetched clients", "count", len(clients))

	if len(clients) == 0 {
		log.Fatal("no clients found")
	}

	// Get first client ID
	clientID := clients[0].ClientID
	logger.Info("testing edit on client", "id", clientID)

	// Test EntityService directly
	svc := clientmod.New(api)
	state, err := svc.Fetch(ctx, clientID)
	if err != nil {
		logger.Error("fetch failed", "error", err)
		log.Fatal(err)
	}
	logger.Info("fetch succeeded", "name", state["name"])

	// Now test the EditOverlay
	historyDir := "/Users/sam/.a0hero/history"
	os.MkdirAll(historyDir, 0755)

	cfg2 := views.EditOverlayConfig{
		EntityType: "client",
		EntityID:   clientID,
		Fields:     clientmod.ClientFields,
		Service:    svc,
		OnClose:    func() tea.Msg { return nil },
		HistoryDir: historyDir,
	}

	logger.Info("creating edit overlay")
	overlay, cmd := views.NewEditOverlay(cfg2)
	if overlay == nil {
		log.Fatal("overlay is nil!")
	}
	logger.Info("overlay created", "has_cmd", cmd != nil)

	// Simulate WindowSizeMsg
	overlay.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	logger.Info("WindowSizeMsg sent to overlay", "overlay_loading", overlay.Loading())

	// Run the fetch command
	if cmd != nil {
		logger.Info("running fetch command")
		msg := cmd()
		logger.Info("fetch command returned", "msg_type", fmt.Sprintf("%T", msg))

		switch m := msg.(type) {
		case views.EditOverlayReady:
			logger.Info("EditOverlayReady received!", "has_session", m.Session != nil)
			overlay.HandleReady(m.Session)
		case views.EditOverlayError:
			logger.Error("EditOverlayError received", "message", m.Message)
		default:
			logger.Warn("unexpected message type", "type", fmt.Sprintf("%T", msg))
		}
	}

	// Check if overlay is ready
	logger.Info("final state", "loading", overlay.Loading(), "has_session", overlay.GetSession() != nil, "error_count", len(overlay.GetErrors()))

	if overlay.Loading() {
		log.Fatal("overlay is still loading after fetch completed!")
	}

	if overlay.GetSession() == nil {
		log.Fatal("overlay has no session!")
	}

	// Get the view
	view := overlay.View()
	logger.Info("overlay view rendered", "length", len(view), "has_content", len(view) > 20)

	if len(view) < 50 {
		log.Fatal("view seems too short:", view)
	}

	// Simulate what App.View() does
	appView := fmt.Sprintf("\nTab bar\n\n✓ Editing: %s\n\n%s\n\nesc: close\n",
		overlay.EntityID(), view)
	logger.Info("App View simulation", "length", len(appView))

	logger.Info("TEST PASSED - EditOverlay works correctly!")
}
