package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/samrocksc/a0hero/logger"
	"github.com/samrocksc/a0hero/tui"
)

var (
	configDir string
	debug     bool
)

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
			configDir = filepath.Join(os.Getenv("HOME"), ".config", "a0hero")
		}
	}

	rootCmd.Flags().StringVar(&configDir, "config-dir", configDir, "directory for tenant config files")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "enable debug logging to logs/")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI() error {
	// Set up logger first (before anything else)
	if err := logger.Setup(debug, "logs"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to set up logger: %v\n", err)
	}
	defer logger.Close()

	if debug {
		fmt.Fprintf(os.Stderr, "debug logs → %s\n", logger.LogPath())
	}

	// Ensure config dir exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	logger.Info("starting a0hero", "config_dir", configDir, "debug", debug)

	app := tui.NewApp(configDir, debug)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	logger.Info("a0hero exited", "error", err)
	return err
}