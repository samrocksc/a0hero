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
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/samrocksc/a0hero/client"
	clientmod "github.com/samrocksc/a0hero/modules/clients"
	connmod "github.com/samrocksc/a0hero/modules/connections"
	logmod "github.com/samrocksc/a0hero/modules/logs"
	rolemod "github.com/samrocksc/a0hero/modules/roles"
	usermod "github.com/samrocksc/a0hero/modules/users"

	"github.com/samrocksc/a0hero/logger"
	"github.com/samrocksc/a0hero/tui/components"
	"github.com/samrocksc/a0hero/tui/views"
)

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	brandBg = lipgloss.Color("#7C58CB")

	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(brandBg).
			Bold(true).
			Padding(0, 2)

	tabGapStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(brandBg).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(brandBg)

	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	colHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))

	detailBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#555555")).
				Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))
)

// ---------------------------------------------------------------------------
// Tabs / Sections
// ---------------------------------------------------------------------------

type section int

const (
	secUsers section = iota
	secClients
	secRoles
	secConnections
	secLogs
	secConfigure
	secCount // sentinel
)

var sectionNames = [secCount]string{
	"Users",
	"Clients",
	"Roles",
	"Connections",
	"Logs",
	"Configure",
}

// Configure sub-menu items
type configItem int

const (
	cfgModifyCurrent configItem = iota
	cfgAddTenant
	cfgExit
	cfgCount
)

var configItemNames = [cfgCount]string{
	"Modify Current",
	"Add Tenant",
	"Exit",
}

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

// Cache entry for section data
type cacheEntry struct {
	items    []moduleItem
	cols     []string
	expiresAt time.Time
}

// App is the root Bubble Tea model.
type App struct {
	configDir string
	cfg       *client.Config
	api       *client.Client

	// Current section
	section  section
	previous section

	// Module content
	items        []moduleItem
	cursor       int
	cols         []string
	loading      bool
	err          string

	// Cache for section data (30 seconds)
	cache     map[section]*cacheEntry
	cacheTTL  time.Duration

	// Loading context for cancellation
	fetchCtx    context.Context
	fetchCancel context.CancelFunc
	fetchTimeout time.Duration

	// Detail overlay
	showDetail   bool
	detailContent string

	// Edit overlay (for inline editing)
	editOverlay   *views.EditOverlay

	// Configure sub-menu
	configCursor    configItem
	configItems     []string
	configForm     *huh.Form
	configName     string
	configDomain   string
	configCID      string
	configSecret   string
	configEditing  bool // true = editing existing, false = adding new

	// Connection state
	tenant  string
	domain  string
	connected bool

	// Spinner
	spinner spinner.Model

	// Viewport for detail/scroll
	viewport viewport.Model

	// Terminal size
	width  int
	height int

	// Debug
	debug bool
}

// NewApp creates a new App model.
func NewApp(configDir string, debug bool) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C58CB"))

	return &App{
		configDir:    configDir,
		section:      secUsers,
		spinner:       s,
		debug:         debug,
		loading:       true,
		fetchTimeout:  10 * time.Second,
		cache:         make(map[section]*cacheEntry),
		cacheTTL:      30 * time.Second,
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
		logger.Info("loading config", "config_dir", a.configDir)

		clientID := os.Getenv("AUTH0_CLIENT_ID")
		clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
		domain := os.Getenv("AUTH0_DOMAIN")

		if clientID != "" && clientSecret != "" && domain != "" {
			logger.Info("using env vars for auth", "domain", domain)
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

		tenants, err := client.AvailableTenants(a.configDir)
		if err != nil || len(tenants) == 0 {
			logger.Warn("no tenants found in config dir")
			return authenticated{err: fmt.Errorf("no tenants configured")}
		}

		logger.Info("loading tenant config", "tenant", tenants[0])
		cfg, err := client.Load(filepath.Join(a.configDir, tenants[0]))
		if err != nil {
			return authenticated{err: err}
		}

		c, err := client.NewClientFromConfig(cfg)
		if err != nil {
			return authenticated{client: nil, cfg: cfg, err: err}
		}
		logger.Info("connected to tenant", "tenant", cfg.Name, "domain", cfg.Domain)
		return authenticated{client: c, cfg: cfg}
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Let the spinner tick
	s, spinCmd := a.spinner.Update(msg)
	a.spinner = s
	if spinCmd != nil {
		cmds = append(cmds, spinCmd)
	}

	// Let the viewport scroll
	if a.viewport.Width > 0 {
		v, vpCmd := a.viewport.Update(msg)
		a.viewport = v
		if vpCmd != nil {
			cmds = append(cmds, vpCmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.viewport.Width = msg.Width
		a.viewport.Height = a.contentHeight()
		cmds = append(cmds, a.fetchCurrentSection())
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		return a.handleKey(msg)

	case authenticated:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err.Error()
			logger.Error("authentication failed", "error", msg.err)
			if msg.cfg != nil {
				a.tenant = msg.cfg.Name
				a.domain = msg.cfg.Domain
			}
			a.connected = false
			a.section = secConfigure
			a.configForm = nil; a.buildConfigMenu()
			return a, tea.Batch(cmds...)
		}
		a.api = msg.client
		a.tenant = msg.cfg.Name
		a.domain = msg.cfg.Domain
		a.connected = true
		a.err = ""
		cmds = append(cmds, a.fetchCurrentSection())
		return a, tea.Batch(cmds...)

	case moduleItemsMsg:
		a.loading = false
		if msg.err != nil {
			a.err = msg.err.Error()
			logger.Error("module fetch failed", "section", sectionNames[a.section], "error", msg.err)
			return a, tea.Batch(cmds...)
		}
		a.items = msg.items
		a.cursor = 0
		a.err = ""
		a.showDetail = false
		// Save to cache
		a.saveToCache(a.section, msg.items, a.cols)
		logger.Info("module data loaded", "section", sectionNames[a.section], "count", len(msg.items))
		return a, tea.Batch(cmds...)

	case configDoneMsg:
		if msg.err != nil {
			a.err = msg.err.Error()
			logger.Error("configure failed", "error", msg.err)
			a.configForm = nil
			return a, tea.Batch(cmds...)
		}
		a.api = msg.api
		a.cfg = msg.cfg
		a.tenant = msg.cfg.Name
		a.domain = msg.cfg.Domain
		a.connected = true
		a.err = ""
		a.configForm = nil
		a.configEditing = false
		a.buildConfigMenu()
		logger.Info("configure success", "tenant", msg.cfg.Name, "domain", msg.cfg.Domain)
		a.section = secUsers
		cmds = append(cmds, a.fetchCurrentSection())
		return a, tea.Batch(cmds...)

	case views.EditOverlayReady, views.EditOverlayError, views.EditOverlaySaved:
		// Forward edit overlay messages to the overlay
		if a.editOverlay != nil {
			updated, cmd := a.editOverlay.Update(msg)
			a.editOverlay = updated.(*views.EditOverlay)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)

	case EditOverlayClosed:
		// Edit overlay was closed
		a.editOverlay = nil
		return a, nil
	}

	// If we have an active edit overlay, forward all messages to it
	if a.editOverlay != nil {
		updated, cmd := a.editOverlay.Update(msg)
		a.editOverlay = updated.(*views.EditOverlay)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	// Forward to config form if active
	if a.section == secConfigure && a.configForm != nil {
		_, cmd := a.configForm.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Check if form completed after any update
		if a.configForm.State == huh.StateCompleted {
			cmds = append(cmds, a.submitConfig())
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Detail overlay mode
	if a.showDetail {
		switch msg.String() {
		case "esc", "q":
			a.showDetail = false
			return a, nil
		}
		// Let viewport handle scrolling
		return a, nil
	}

	// Configure sub-menu — navigate items or handle form
	if a.section == secConfigure && a.configForm != nil {
		switch msg.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "esc":
			// esc in form = go back to submenu
			a.configForm = nil
			a.configEditing = false
			return a, nil
		case "tab", "right", "l":
			// Tab out of form = move to next section
			a.configForm = nil
			a.configEditing = false
			a.section = (a.section + 1) % secCount
			a.cursor = 0
			a.showDetail = false
			a.err = ""
			if a.section == secConfigure {
				a.buildConfigMenu()
			} else {
				cmds = append(cmds, a.fetchCurrentSection())
			}
			return a, tea.Batch(cmds...)
		case "shift+tab", "left", "h":
			// Shift+tab out of form = move to previous section
			a.configForm = nil
			a.configEditing = false
			a.section = (a.section - 1 + secCount) % secCount
			a.cursor = 0
			a.showDetail = false
			a.err = ""
			if a.section == secConfigure {
				a.buildConfigMenu()
			} else {
				cmds = append(cmds, a.fetchCurrentSection())
			}
			return a, tea.Batch(cmds...)
		default:
			_, cmd := a.configForm.Update(msg)
			if a.configForm.State == huh.StateCompleted {
				cmds = append(cmds, a.submitConfig())
			}
			return a, tea.Batch(append(cmds, cmd)...)
		}
	}

	// Configure sub-menu — select item
	if a.section == secConfigure && a.configForm == nil {
		switch msg.String() {
		case "down", "j":
			if a.configCursor < configItem(len(a.configItems))-1 {
				a.configCursor++
			}
		case "up", "k":
			if a.configCursor > 0 {
				a.configCursor--
			}
		case "tab", "right", "l":
			// Tab out of configure menu
			a.section = (a.section + 1) % secCount
			a.cursor = 0
			a.configCursor = 0
			a.err = ""
			if a.section == secConfigure {
				a.configForm = nil
				a.configEditing = false
				a.buildConfigMenu()
			} else {
				cmds = append(cmds, a.fetchCurrentSection())
			}
			return a, tea.Batch(cmds...)
		case "shift+tab", "left", "h":
			// Shift+tab out of configure menu
			a.section = (a.section - 1 + secCount) % secCount
			a.cursor = 0
			a.configCursor = 0
			a.err = ""
			if a.section == secConfigure {
				a.configForm = nil
				a.configEditing = false
				a.buildConfigMenu()
			} else {
				cmds = append(cmds, a.fetchCurrentSection())
			}
			return a, tea.Batch(cmds...)
		case "enter":
			switch a.configCursor {
			case cfgModifyCurrent: // Modify current
				if a.cfg != nil {
					a.configEditing = true
					a.configName = a.cfg.Name
					a.configDomain = a.cfg.Domain
					a.configCID = a.cfg.ClientID
					a.configSecret = a.cfg.ClientSecret
					a.newConfigForm()
					return a, a.configForm.Init()
				} else {
				a.err = "no tenant connected"
				}
			case cfgAddTenant: // Add tenant
				a.configEditing = false
				a.configName = ""
				a.configDomain = ""
				a.configCID = ""
				a.configSecret = ""
				a.newConfigForm()
				return a, a.configForm.Init()
			case cfgExit: // Quit
				return a, tea.Quit
			}
		case "esc", "q":
			return a, tea.Quit
		}
		return a, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return a, tea.Quit

	case "tab", "right", "l":
		a.cancelFetch()
		a.section = (a.section + 1) % secCount
		a.cursor = 0
		a.showDetail = false
		a.err = ""
		if a.section == secConfigure {
			a.configForm = nil
			a.configEditing = false
			a.buildConfigMenu()
		} else {
			cmds = append(cmds, a.fetchCurrentSection())
		}
		return a, tea.Batch(cmds...)

	case "shift+tab", "left", "h":
		a.cancelFetch()
		a.section = (a.section - 1 + secCount) % secCount
		a.cursor = 0
		a.showDetail = false
		a.err = ""
		if a.section == secConfigure {
			a.configForm = nil
			a.configEditing = false
			a.buildConfigMenu()
		} else {
			cmds = append(cmds, a.fetchCurrentSection())
		}
		return a, tea.Batch(cmds...)

	case "down", "j":
		if a.items != nil && a.cursor < len(a.items)-1 {
			a.cursor++
		}
	case "up", "k":
		if a.cursor > 0 {
			a.cursor--
		}

	case "e":
		// Open edit overlay for selected item
		if a.items != nil && a.cursor < len(a.items) && a.api != nil {
			return a.openEditOverlay()
		}

	case "enter":
		if a.items != nil && a.cursor < len(a.items) {
			item := a.items[a.cursor]
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
			a.showDetail = true
			a.viewport.SetContent(a.detailContent)
			a.viewport.GotoTop()
		}
	}

	return a, tea.Batch(cmds...)
}

// ---------------------------------------------------------------------------
// Section switching
// ---------------------------------------------------------------------------

// cancelFetch cancels any in-progress fetch.
func (a *App) cancelFetch() {
	if a.fetchCancel != nil {
		a.fetchCancel()
		a.fetchCancel = nil
	}
	a.loading = false
}

func (a *App) fetchCurrentSection() tea.Cmd {
	// Cancel any existing fetch
	a.cancelFetch()

	// Check cache first
	if cached := a.getFromCache(a.section); cached != nil {
		a.items = cached.items
		a.cols = cached.cols
		a.cursor = 0
		a.loading = false
		a.err = ""
		return nil
	}

	// Create new context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), a.fetchTimeout)
	a.fetchCtx = ctx
	a.fetchCancel = cancel

	if a.api == nil {
		return nil
	}
	switch a.section {
	case secUsers:
		a.cols = usermod.Columns()
		a.loading = true
		return a.fetchUsers(ctx)
	case secClients:
		a.cols = clientmod.Columns()
		a.loading = true
		return a.fetchClients(ctx)
	case secRoles:
		a.cols = rolemod.Columns()
		a.loading = true
		return a.fetchRoles(ctx)
	case secConnections:
		a.cols = connmod.Columns()
		a.loading = true
		return a.fetchConnections(ctx)
	case secLogs:
		a.cols = logmod.Columns()
		a.loading = true
		return a.fetchLogs(ctx)
	}
	return nil
}

// getFromCache returns cached data if still valid.
func (a *App) getFromCache(sec section) *cacheEntry {
	if entry, ok := a.cache[sec]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry
		}
		delete(a.cache, sec)
	}
	return nil
}

// saveToCache saves data to cache.
func (a *App) saveToCache(sec section, items []moduleItem, cols []string) {
	a.cache[sec] = &cacheEntry{
		items:    items,
		cols:     cols,
		expiresAt: time.Now().Add(a.cacheTTL),
	}
}

// ---------------------------------------------------------------------------
// Data fetchers
// ---------------------------------------------------------------------------

func (a *App) fetchUsers(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return moduleItemsMsg{err: fmt.Errorf("request cancelled or timed out")}
		default:
		}
		u := usermod.New(a.api)
		result, err := u.List(ctx, 0, 50)
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

func (a *App) fetchClients(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return moduleItemsMsg{err: fmt.Errorf("request cancelled or timed out")}
		default:
		}
		c := clientmod.New(a.api)
		result, err := c.List(ctx)
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

func (a *App) fetchRoles(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return moduleItemsMsg{err: fmt.Errorf("request cancelled or timed out")}
		default:
		}
		r := rolemod.New(a.api)
		result, err := r.List(ctx)
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

func (a *App) fetchConnections(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return moduleItemsMsg{err: fmt.Errorf("request cancelled or timed out")}
		default:
		}
		c := connmod.New(a.api)
		result, err := c.List(ctx)
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

func (a *App) fetchLogs(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return moduleItemsMsg{err: fmt.Errorf("request cancelled or timed out")}
		default:
		}
		l := logmod.New(a.api)
		result, err := l.List(ctx, "", 50)
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
// Edit Overlay
// -----------------------------------------------------------------------------

// openEditOverlay opens the edit overlay for the currently selected item.
func (a *App) openEditOverlay() (tea.Model, tea.Cmd) {
	if len(a.items) == 0 || a.cursor >= len(a.items) {
		return a, nil
	}

	item := a.items[a.cursor]

	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".a0hero", "history")

	// Determine entity type and create service based on current section
	var cfg views.EditOverlayConfig

	switch a.section {
	case secClients:
		clientSvc := clientmod.New(a.api)
		cfg = views.EditOverlayConfig{
			EntityType: "client",
			EntityID:   item.id,
			Fields:     clientmod.ClientFields,
			Service:    clientSvc,
			OnClose:    func() tea.Msg { return EditOverlayClosed{} },
			HistoryDir: historyDir,
		}
	default:
		a.err = "editing not supported for this section"
		return a, nil
	}

	var cmd tea.Cmd
	a.editOverlay, cmd = views.NewEditOverlay(cfg)
	return a, cmd
}

// Message for edit overlay closed
type EditOverlayClosed struct{}

// ---------------------------------------------------------------------------
// Configure
// ---------------------------------------------------------------------------

func (a *App) newConfigForm() {
	a.configForm = huh.NewForm(
		huh.NewGroup(
		huh.NewInput().Title("Tenant Name").Value(&a.configName).Placeholder("dev"),
		huh.NewInput().Title("Domain").Value(&a.configDomain).Placeholder("dev-tenant.auth0app.com"),
		huh.NewInput().Title("Client ID").Value(&a.configCID).Placeholder("your-client-id"),
		huh.NewInput().Title("Client Secret").Value(&a.configSecret).Placeholder("your-client-secret").EchoMode(huh.EchoModePassword),
		),
	).WithWidth(50)
}

// buildConfigMenu builds the list of configure options based on current state.
func (a *App) buildConfigMenu() {
	a.configItems = []string{}
	if a.cfg != nil {
		a.configItems = append(a.configItems, fmt.Sprintf("Modify Current: %s (%s)", a.tenant, a.domain))
	} else {
		a.configItems = append(a.configItems, "Modify Current (not connected)")
	}
	a.configItems = append(a.configItems, "Add Tenant")
	a.configItems = append(a.configItems, "Exit")
	a.configCursor = 0
}

func (a *App) submitConfig() tea.Cmd {
	name := a.configName
	domain := a.configDomain
	cid := a.configCID
	secret := a.configSecret
	configDir := a.configDir

	return func() tea.Msg {
		logger.Info("submitting config", "tenant", name, "domain", domain)

		if name == "" || domain == "" || cid == "" || secret == "" {
			return configDoneMsg{err: fmt.Errorf("all fields are required")}
		}

		cfg := &client.Config{
			Name:         name,
			Domain:       domain,
			ClientID:     cid,
			ClientSecret: secret,
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return configDoneMsg{err: fmt.Errorf("create config dir: %w", err)}
		}
		configPath := filepath.Join(configDir, name+".yaml")
		if err := cfg.WriteFile(configPath); err != nil {
			return configDoneMsg{err: fmt.Errorf("write config: %w", err)}
		}
		logger.Info("config file written", "path", configPath)

		c, err := client.NewClientFromConfig(cfg)
		if err != nil {
			return configDoneMsg{err: fmt.Errorf("connection failed: %w", err)}
		}

		return configDoneMsg{cfg: cfg, api: c}
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

func (a *App) contentHeight() int {
	// 1 tab bar + 1 info bar + 1 divider + 1 help bar = 4 lines overhead
	h := a.height - 4
	if h < 5 {
		h = 5
	}
	return h
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Still connecting
	if a.loading && a.api == nil && a.err == "" {
		return fmt.Sprintf("\n  %s Connecting to Auth0...\n", a.spinner.View())
	}

	// Render edit overlay if active
	if a.editOverlay != nil {
		return a.renderEditOverlay()
	}

	var b strings.Builder

	// 1. Tab bar
	b.WriteString(a.renderTabs())
	b.WriteString("\n")

	// 2. Info bar (tenant + connection status)
	b.WriteString(a.renderInfoBar())
	b.WriteString("\n")

	// 3. Divider
	b.WriteString(dividerStyle.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	// 4. Content area (or detail overlay)
	if a.showDetail {
		b.WriteString(a.renderDetail())
	} else {
		b.WriteString(a.renderContent())
	}

	// 5. Help bar
	b.WriteString("\n")
	b.WriteString(a.renderHelp())

	return b.String()
}

// renderEditOverlay renders the edit overlay content.
func (a *App) renderEditOverlay() string {
	var b strings.Builder

	// Tab bar (same as normal)
	b.WriteString(a.renderTabs())
	b.WriteString("\n")

	// Info bar
	status := successStyle.Render("✓ Editing: " + a.editOverlay.EntityID())
	b.WriteString(status)
	b.WriteString("\n")

	// Divider
	b.WriteString(dividerStyle.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	// Edit overlay content
	b.WriteString(a.editOverlay.View())

	// Help bar
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(" esc: close"))

	return b.String()
}

func (a *App) renderTabs() string {
	var tabs []string
	for i, name := range sectionNames {
		if section(i) == a.section {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}

	// Join with a subtle separator
	tabRow := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)

	// Right-align the brand
	brand := titleStyle.Render(" A0Hero ")

	// Calculate padding between tabs and brand
	contentWidth := lipgloss.Width(tabRow) + lipgloss.Width(brand)
	if contentWidth < a.width {
		gap := a.width - contentWidth
		tabRow = tabRow + tabGapStyle.Render(strings.Repeat(" ", gap))
	}

	// This puts brand at far right
	// Actually let's just do: brand | tabs
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, brand, tabRow)

	return row
}

func (a *App) renderInfoBar() string {
	if a.connected {
		status := successStyle.Render("✓ " + a.tenant)
		domain := headerStyle.Render(a.domain)
		right := headerStyle.Render(fmt.Sprintf("  %s", domain))
		return lipgloss.JoinHorizontal(lipgloss.Bottom, status, right)
	}
	if a.err != "" {
		return errorStyle.Render("⚠  " + a.err)
	}
	return headerStyle.Render("Not connected")
}

func (a *App) renderContent() string {
	if a.section == secConfigure {
		return a.renderConfigure()
	}

	if !a.connected {
		return "\n  Not connected — tab to Configure to set up a tenant.\n"
	}

	if a.loading {
		return fmt.Sprintf("\n  %s Loading %s...\n", a.spinner.View(), sectionNames[a.section])
	}

	if a.err != "" {
		return errorStyle.Render(fmt.Sprintf("\n  error: %s\n", a.err))
	}

	if len(a.items) == 0 {
		return normalRowStyle.Render(fmt.Sprintf("\n  No %s found.\n", strings.ToLower(sectionNames[a.section])))
	}

	return a.renderTable()
}

func (a *App) renderTable() string {
	rows := make([][]string, len(a.items))
	for i, item := range a.items {
		rows[i] = item.cols
	}

	maxRows := a.contentHeight() - 2
	if maxRows < 3 {
		maxRows = 20
	}

	// Determine visible window (scroll)
	start := 0
	end := len(rows)
	if end-start > maxRows {
		half := maxRows / 2
		start = a.cursor - half
		if start < 0 {
			start = 0
		}
		end = start + maxRows
		if end > len(rows) {
			end = len(rows)
			start = end - maxRows
			if start < 0 {
				start = 0
			}
		}
	}

	visRows := rows[start:end]

	tbl := components.NewTable(a.cols).
		WithRows(visRows).
		WithSelected(a.cursor - start).
		WithWidth(a.width)

	return tbl.Render()
}

func (a *App) renderDetail() string {
	return detailBorderStyle.Render(a.detailContent)
}

func (a *App) renderConfigure() string {
	var b strings.Builder
	b.WriteString("\n")

	if a.configForm != nil {
		// Form is active
		if a.configEditing {
			b.WriteString(activeTabStyle.Render("  Editing: " + a.configName))
		} else {
			b.WriteString(activeTabStyle.Render("  Add New Tenant"))
		}
		b.WriteString("\n\n")
		b.WriteString(a.configForm.View())
		return b.String()
	}

	// Sub-menu
	b.WriteString(colHeaderStyle.Render("  Configure"))
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 30)))
	b.WriteString("\n")

	for i, item := range a.configItems {
		if i == int(a.configCursor) {
			b.WriteString(selectedRowStyle.Render("  ➤ " + item))
		} else {
			b.WriteString(normalRowStyle.Render("    " + item))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (a *App) renderHelp() string {
	if a.showDetail {
		return helpStyle.Render(" esc/back close detail  •  ↑↓ scroll  •  q quit")
	}
	if a.section == secConfigure {
		if a.configForm != nil {
			return helpStyle.Render(" esc back to menu  •  tab/h← switch section  •  enter submit")
		}
		return helpStyle.Render(" tab/h← switch section  •  ↑↓ choose  •  enter select  •  q quit")
	}
	return helpStyle.Render(fmt.Sprintf(
		" ←/h →/l tab switch  •  ↑/k ↓/j scroll %s  •  enter detail  •  e edit  •  q quit",
		f.dimCount(a.cursor, len(a.items)),
	))
}

// small helper for help line
type fmtHelper struct{}

var f = fmtHelper{}

func (fmtHelper) dimCount(cursor, total int) string {
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("(%d/%d)", cursor+1, total)
}