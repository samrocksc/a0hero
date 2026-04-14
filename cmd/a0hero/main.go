package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/samrocksc/a0hero/tui"
)

var configDir string

func main() {
	rootCmd := &cobra.Command{
		Use:   "a0hero",
		Short: "Auth0 tenant manager in your terminal",
		Long:  "A0Hero wraps the Auth0 Management API in a terminal interface for day-to-day administration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	// Default config dir: ~/.config/a0hero/ or XDG_CONFIG_HOME
	if configDir == "" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			configDir = filepath.Join(xdg, "a0hero")
		} else {
			home := os.Getenv("HOME")
			configDir = filepath.Join(home, ".config", "a0hero")
		}
	}

	rootCmd.Flags().StringVar(&configDir, "config-dir", configDir, "directory for tenant config files")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI() error {
	// Ensure config dir exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	app := tui.NewApp(configDir)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	return err
}