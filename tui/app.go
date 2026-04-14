// Package tui implements the Bubble Tea TUI for a0hero.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/samrocksc/a0hero/client"
	clientmod "github.com/samrocksc/a0hero/modules/clients"
	connmod "github.com/samrocksc/a0hero/modules/connections"
	logmod "github.com/samrocksc/a0hero/modules/logs"
	rolemod "github.com/samrocksc/a0hero/modules/roles"
	usermod "github.com/samrocksc/a0hero/modules/users"
)

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7C58CB")).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C58CB"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C58CB")).
			Padding(0, 1)
)

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

type view int

const (
	viewMainMenu view = iota
	viewConfigure
	viewModule
	viewDetail
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type authenticated struct {
	client *client.Client
	cfg    *client.Config
	err    error
}

type moduleItemsMsg struct {
	items []moduleItem
	err   error
}

type configDoneMsg struct {
	cfg *client.Config
	api *client.Client
	err error
}

type moduleItem struct {
	id   string
	cols []string
	dict map[string]string
}

// ---------------------------------------------------------------------------
// App model
// ---------------------------------------------------------------------------

// App is the root Bubble Tea model.
type App struct {
	configDir string
	cfg       *client.Config
	api       *client.Client

	view     view
	previous view

	// Main menu
	menuIndex int
	menuItems []string

	// Module view
	moduleItems   []moduleItem
	moduleIndex   int
	moduleCols    []string
	moduleTitle   string
	moduleLoading bool

	// Detail view
	detailContent string

	// Configure form — values stored here so they survive across updates
	configForm    *huh.Form
	configName    string
	configDomain  string
	configCID     string
	configSecret  string

	// Status
	loading bool
	spinner spinner.Model
	err     string
	tenant  string
	domain  string
	width   int
	height  int
}

// NewApp creates a new App model.
func NewApp(configDir string) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C58CB"))

	return &App{
		configDir: configDir,
		view:      viewMainMenu,
		menuItems: []string{"Users", "Clients (Applications)", "Roles", "Connections", "Logs", "Configure", "Quit"},
		spinner:   s,
		loading:   true,
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.spinner.Tick, a.loadConfig())
}

func (a *App) loadConfig() tea.Cmd {
	return func() tea.Msg {
		// Check for env vars first
		clientID := os.Getenv("AUTH0_CLIENT_ID")
		clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
		domain := os.Getenv("AUTH0_DOMAIN")

		if clientID != "" && clientSecret != "" && domain != "" {
			cfg := &client.Config{
				Name:         "env",
				Domain:       domain,
				ClientID:     clientID,
				ClientSecret: clientSecret,
			}
			c, err := client.NewClientFromConfig(cfg)
			if err != nil {
				return authenticated{err: err}
			}
			return authenticated{client: c, cfg: cfg}
		}

		// Try to load from config dir
		tenants, err := client.AvailableTenants(a.configDir)
		if err != nil || len(tenants) == 0 {
			return authenticated{err: fmt.Errorf("no tenants configured")}
		}

		cfg, err := client.Load(filepath.Join(a.configDir, tenants[0]))
		if err != nil {
			return authenticated{err: err}
		}

		c, err := client.NewClientFromConfig(cfg)
		if err != nil {
			return authenticated{client: nil, cfg: cfg, err: err}
		}
		return authenticated{client: c, cfg: cfg}
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		if a.view == viewConfigure && a.configForm != nil {
			switch msg.String() {
			case "ctrl+c":
				return a, tea.Quit
			case "esc":
				a.view = viewMainMenu
				a.configForm = nil
				return a, nil
			}
			// Forward to huh; after each key, check if form completed
			_, cmd := a.configForm.Update(msg)
			if a.configForm.State == huh.StateCompleted {
				return a, a.submitConfig()
			}
			return a, cmd
		}
		return a.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case authenticated:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err.Error()
			if strings.Contains(a.err, "no tenants") {
				a.view = viewConfigure
				a.newConfigForm()
				return a, a.configForm.Init()
			}
			if msg.cfg != nil {
				a.tenant = msg.cfg.Name
				a.domain = msg.cfg.Domain
			}
			a.api = msg.client
			return a, nil
		}
		a.api = msg.client
		a.tenant = msg.cfg.Name
		a.domain = msg.cfg.Domain
		a.err = ""
		return a, nil

	case moduleItemsMsg:
		a.moduleLoading = false
		if msg.err != nil {
			a.err = msg.err.Error()
			return a, nil
		}
		a.moduleItems = msg.items
		return a, nil

	case configDoneMsg:
		if msg.err != nil {
			a.err = msg.err.Error()
			a.configForm = nil
			a.view = viewMainMenu
			return a, nil
		}
		a.api = msg.api
		a.cfg = msg.cfg
		a.tenant = msg.cfg.Name
		a.domain = msg.cfg.Domain
		a.err = ""
		a.configForm = nil
		a.view = viewMainMenu
		return a, nil
	}

	// Forward non-key messages to config form if active (spinner ticks, etc.)
	if a.view == viewConfigure && a.configForm != nil {
		_, cmd := a.configForm.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return a, tea.Quit

	case "q":
		if a.view == viewMainMenu {
			return a, tea.Quit
		}
		a.view = viewMainMenu
		a.moduleItems = nil
		a.detailContent = ""
		a.err = ""
		a.moduleIndex = 0
		return a, nil

	case "esc":
		if a.view == viewDetail {
			a.view = a.previous
			a.detailContent = ""
			return a, nil
		}
		if a.view == viewModule {
			a.view = viewMainMenu
			a.moduleItems = nil
			a.err = ""
			return a, nil
		}
		return a, tea.Quit

	case "up", "k":
		if a.view == viewMainMenu {
			if a.menuIndex > 0 {
				a.menuIndex--
			}
		} else if a.view == viewModule && a.moduleItems != nil {
			if a.moduleIndex > 0 {
				a.moduleIndex--
			}
		}

	case "down", "j":
		if a.view == viewMainMenu {
			if a.menuIndex < len(a.menuItems)-1 {
				a.menuIndex++
			}
		} else if a.view == viewModule && a.moduleItems != nil {
			if a.moduleIndex < len(a.moduleItems)-1 {
				a.moduleIndex++
			}
		}

	case "enter":
		return a.handleEnter()
	}

	return a, nil
}

func (a *App) handleEnter() (tea.Model, tea.Cmd) {
	switch a.view {
	case viewMainMenu:
		return a.selectMenu()
	case viewModule:
		if a.moduleItems != nil && a.moduleIndex < len(a.moduleItems) {
			item := a.moduleItems[a.moduleIndex]
			if item.dict != nil {
				var b strings.Builder
				maxKeyLen := 0
				for k := range item.dict {
					if len(k) > maxKeyLen {
						maxKeyLen = len(k)
					}
				}
				for k, v := range item.dict {
					padding := strings.Repeat(" ", maxKeyLen-len(k)+1)
					fmt.Fprintf(&b, "%s:%s%s\n", k, padding, v)
				}
				a.detailContent = b.String()
			} else {
				a.detailContent = strings.Join(item.cols, " | ")
			}
			a.previous = viewModule
			a.view = viewDetail
		}
		return a, nil
	}
	return a, nil
}

func (a *App) selectMenu() (tea.Model, tea.Cmd) {
	a.moduleIndex = 0
	a.moduleItems = nil
	a.err = ""

	if a.api == nil && a.menuItems[a.menuIndex] != "Configure" && a.menuItems[a.menuIndex] != "Quit" {
		a.err = "not connected — configure a tenant first"
		return a, nil
	}

	switch a.menuItems[a.menuIndex] {
	case "Users":
		a.view = viewModule
		a.moduleTitle = "Users"
		a.moduleCols = usermod.Columns()
		a.moduleLoading = true
		return a, a.fetchUsers()
	case "Clients (Applications)":
		a.view = viewModule
		a.moduleTitle = "Clients"
		a.moduleCols = clientmod.Columns()
		a.moduleLoading = true
		return a, a.fetchClients()
	case "Roles":
		a.view = viewModule
		a.moduleTitle = "Roles"
		a.moduleCols = rolemod.Columns()
		a.moduleLoading = true
		return a, a.fetchRoles()
	case "Connections":
		a.view = viewModule
		a.moduleTitle = "Connections"
		a.moduleCols = connmod.Columns()
		a.moduleLoading = true
		return a, a.fetchConnections()
	case "Logs":
		a.view = viewModule
		a.moduleTitle = "Logs"
		a.moduleCols = logmod.Columns()
		a.moduleLoading = true
		return a, a.fetchLogs()
	case "Configure":
		a.view = viewConfigure
		a.newConfigForm()
		return a, a.configForm.Init()
	case "Quit":
		return a, tea.Quit
	}
	return a, nil
}

// ---------------------------------------------------------------------------
// Configure form
// ---------------------------------------------------------------------------

func (a *App) newConfigForm() {
	// Reset values
	a.configName = ""
	a.configDomain = ""
	a.configCID = ""
	a.configSecret = ""

	a.configForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Tenant Name").Value(&a.configName).Placeholder("dev"),
			huh.NewInput().Title("Domain").Value(&a.configDomain).Placeholder("dev-tenant.auth0.com"),
			huh.NewInput().Title("Client ID").Value(&a.configCID).Placeholder("your-client-id"),
			huh.NewInput().Title("Client Secret").Value(&a.configSecret).Placeholder("your-client-secret").EchoMode(huh.EchoModePassword),
		),
	).WithWidth(50)
}

// submitConfig is a tea.Cmd that writes the YAML config file, authenticates,
// and returns a configDoneMsg.
func (a *App) submitConfig() tea.Cmd {
	name := a.configName
	domain := a.configDomain
	cid := a.configCID
	secret := a.configSecret
	configDir := a.configDir

	return func() tea.Msg {
		if name == "" || domain == "" || cid == "" || secret == "" {
			return configDoneMsg{err: fmt.Errorf("all fields are required")}
		}

		cfg := &client.Config{
			Name:         name,
			Domain:       domain,
			ClientID:     cid,
			ClientSecret: secret,
		}

		// Write config file
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return configDoneMsg{err: fmt.Errorf("create config dir: %w", err)}
		}
		configPath := filepath.Join(configDir, name+".yaml")
		if err := cfg.WriteFile(configPath); err != nil {
			return configDoneMsg{err: fmt.Errorf("write config: %w", err)}
		}

		// Authenticate
		c, err := client.NewClientFromConfig(cfg)
		if err != nil {
			return configDoneMsg{err: fmt.Errorf("connection failed: %w", err)}
		}

		return configDoneMsg{cfg: cfg, api: c}
	}
}

// ---------------------------------------------------------------------------
// Data fetchers
// ---------------------------------------------------------------------------

func (a *App) fetchUsers() tea.Cmd {
	return func() tea.Msg {
		u := usermod.New(a.api)
		result, err := u.List(context.Background(), 0, 50)
		if err != nil {
			return moduleItemsMsg{err: err}
		}
		items := make([]moduleItem, len(result))
		for i, user := range result {
			items[i] = moduleItem{
				id:   user.ID,
				cols: user.Row(),
				dict: map[string]string{
					"ID":             user.ID,
					"Email":          user.Email,
					"Name":           user.Name,
					"Email Verified":  fmt.Sprintf("%v", user.EmailVerified),
					"Last Login":     formatTimePtr(user.LastLogin),
				},
			}
		}
		return moduleItemsMsg{items: items}
	}
}

func (a *App) fetchClients() tea.Cmd {
	return func() tea.Msg {
		c := clientmod.New(a.api)
		result, err := c.List(context.Background())
		if err != nil {
			return moduleItemsMsg{err: err}
		}
		items := make([]moduleItem, len(result))
		for i, cl := range result {
			items[i] = moduleItem{
				id:   cl.ClientID,
				cols: cl.Row(),
				dict: map[string]string{
					"Client ID":      cl.ClientID,
					"Name":           cl.Name,
					"App Type":       cl.AppType,
					"Description":    cl.Description,
					"Callbacks":      strings.Join(cl.Callbacks, ", "),
					"Redirect URIs":  strings.Join(cl.RedirectURIs, ", "),
				},
			}
		}
		return moduleItemsMsg{items: items}
	}
}

func (a *App) fetchRoles() tea.Cmd {
	return func() tea.Msg {
		r := rolemod.New(a.api)
		result, err := r.List(context.Background())
		if err != nil {
			return moduleItemsMsg{err: err}
		}
		items := make([]moduleItem, len(result))
		for i, role := range result {
			items[i] = moduleItem{
				id:   role.ID,
				cols: role.Row(),
				dict: map[string]string{
					"ID":          role.ID,
					"Name":        role.Name,
					"Description": role.Description,
				},
			}
		}
		return moduleItemsMsg{items: items}
	}
}

func (a *App) fetchConnections() tea.Cmd {
	return func() tea.Msg {
		c := connmod.New(a.api)
		result, err := c.List(context.Background())
		if err != nil {
			return moduleItemsMsg{err: err}
		}
		items := make([]moduleItem, len(result))
		for i, conn := range result {
			items[i] = moduleItem{
				id:   conn.ID,
				cols: conn.Row(),
				dict: map[string]string{
					"ID":              conn.ID,
					"Name":            conn.Name,
					"Strategy":       conn.Strategy,
					"Enabled Clients": fmt.Sprintf("%d clients", len(conn.EnabledClients)),
				},
			}
		}
		return moduleItemsMsg{items: items}
	}
}

func (a *App) fetchLogs() tea.Cmd {
	return func() tea.Msg {
		l := logmod.New(a.api)
		result, err := l.List(context.Background(), "", 50)
		if err != nil {
			return moduleItemsMsg{err: err}
		}
		items := make([]moduleItem, len(result))
		for i, evt := range result {
			items[i] = moduleItem{
				id:   evt.ID,
				cols: evt.Row(),
				dict: map[string]string{
					"Time":      evt.FormatDate(),
					"Type":      evt.Type,
					"Event":     evt.Describe(),
					"User":      evt.UserName,
					"IP":        evt.IP,
					"Client ID": evt.ClientID,
				},
			}
		}
		return moduleItemsMsg{items: items}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("2006-01-02 15:04:05")
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (a *App) View() string {
	if a.width == 0 {
		a.width = 80
	}

	if a.loading {
		return fmt.Sprintf("\n\n  %s Connecting to Auth0...\n\n", a.spinner.View())
	}

	switch a.view {
	case viewMainMenu:
		return a.viewMainMenu()
	case viewConfigure:
		return a.viewConfigure()
	case viewModule:
		return a.viewModule()
	case viewDetail:
		return a.viewDetail()
	default:
		return a.viewMainMenu()
	}
}

func (a *App) viewMainMenu() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" A0Hero "))
	b.WriteString("\n\n")

	if a.api != nil {
		b.WriteString(successStyle.Render("✓ Connected"))
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %s (%s)", a.tenant, a.domain)))
		b.WriteString("\n\n")
	} else if a.err != "" {
		b.WriteString(errorStyle.Render("⚠  Not connected"))
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(a.err))
		b.WriteString("\n\n")
	}

	for i, item := range a.menuItems {
		if i == a.menuIndex {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" ➤ %s", item)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("   %s", item)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k ↓/j navigate • enter select • q quit"))

	return b.String()
}

func (a *App) viewConfigure() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" A0Hero — Configure "))
	b.WriteString("\n\n")

	if a.configForm != nil {
		b.WriteString(a.configForm.View())
	} else {
		b.WriteString("Setting up form...")
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("esc to cancel • tab to move between fields • enter to submit"))

	return b.String()
}

func (a *App) viewModule() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" A0Hero — %s ", a.moduleTitle)))
	b.WriteString("\n\n")

	if a.moduleLoading {
		b.WriteString(fmt.Sprintf("  %s Loading...\n", a.spinner.View()))
		return b.String()
	}

	if a.err != "" {
		b.WriteString(errorStyle.Render(fmt.Sprintf("error: %s", a.err)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("esc to go back"))
		return b.String()
	}

	if len(a.moduleItems) == 0 {
		b.WriteString(normalStyle.Render("No items found."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("esc to go back"))
		return b.String()
	}

	b.WriteString(normalStyle.Bold(true).Render(strings.Join(a.moduleCols, "  ")))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n")

	maxRows := a.height - 10
	if maxRows < 5 {
		maxRows = 20
	}
	end := a.moduleIndex + maxRows/2
	start := a.moduleIndex - maxRows/2
	if start < 0 {
		start = 0
	}
	if end > len(a.moduleItems) {
		end = len(a.moduleItems)
	}
	if end-start > maxRows {
		start = end - maxRows
	}

	for i := start; i < end; i++ {
		item := a.moduleItems[i]
		row := strings.Join(item.cols, "  ")
		if i == a.moduleIndex {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" ➤ %s", row)))
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("   %s", row)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("%d/%d items • ↑/k ↓/j • enter detail • esc back", a.moduleIndex+1, len(a.moduleItems))))

	return b.String()
}

func (a *App) viewDetail() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf(" A0Hero — %s Detail ", a.moduleTitle)))
	b.WriteString("\n\n")

	b.WriteString(borderStyle.Render(a.detailContent))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("esc back • q main menu"))

	return b.String()
}