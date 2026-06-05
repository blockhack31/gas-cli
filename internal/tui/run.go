package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/calghar/gas-cli/internal/config"
)

// Run starts the interactive identity management console.
func Run() error {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	cfg, err := configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	m := newModel(cfg, configMgr)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run terminal UI: %w", err)
	}
	return nil
}
