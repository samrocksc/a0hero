# A0Hero — Architecture

## Overview

A0Hero is a Go TUI application that wraps Auth0's Management API in a terminal interface. The application is structured in layers, with clean separation between the API client, domain modules, and the presentation layer.

The key design goals are:
- **Extensible** — new modules follow the same pattern as existing ones
- **Testable** — all modules are independently testable via mock HTTP handlers
- **Configurable** — tenant context comes from config files, not hardcoded values
- **OpenAPI-driven** — the Auth0 Management API spec is the source of truth for models and behavior

---

## Directory Structure

```
a0hero/
├── AGENTS.md              # Agent onboarding guide (this repo)
├── AGENT_PERSONAS.md      # Persona-specific prompts
├── go.mod
├── go.sum
│
├── cmd/                   # CLI entry points
│   └── a0hero/
│       └── main.go       # Application entry point, cobra commands
│
├── client/                # Auth0 API client (importable independently)
│   ├── client.go          # Auth0 client struct, methods
│   ├── auth.go            # Auth via client credentials / config
│   ├── config.go          # Config file loading (.yaml)
│   └── errors.go          # Shared error types
│
├── models/                # Shared types, interfaces
│   ├── common.go          # Pagination, error wrappers, timestamps
│   └── module.go          # Module interface definition
│
├── modules/               # Domain modules (self-contained, independently testable)
│   ├── users/
│   │   ├── client.go      # Users API calls (list, get, create, update, delete, search)
│   │   ├── model.go       # User type definitions
│   │   └── view.go        # Bubble Tea view for users (optional, may live in tui/)
│   ├── clients/
│   │   ├── client.go      # Clients API calls (list, get, create, update, update redirect_uris)
│   │   ├── model.go       # Client type definitions
│   │   └── view.go
│   ├── roles/
│   │   ├── client.go      # Roles API calls (list, get, create, assign to users)
│   │   ├── model.go
│   │   └── view.go
│   ├── connections/
│   │   ├── client.go      # Connections API calls (list, get, create, update)
│   │   ├── model.go
│   │   └── view.go
│   └── logs/
│       ├── client.go      # Logs API calls (list with date filter, get by event ID)
│       ├── model.go
│       └── view.go
│
├── tui/                   # Bubble Tea UI layer (sits on top of modules/)
│   ├── app.go             # Root Bubble Tea model, main Update/Switch
│   ├── main_menu.go       # Main menu view (Tenant selector, module entry points)
│   ├── tenant.go          # Tenant context and switching
│   ├── views/             # Per-module views
│   │   ├── users.go
│   │   ├── clients.go
│   │   ├── roles.go
│   │   ├── connections.go
│   │   └── logs.go
│   └── components/        # Shared UI components
│       ├── table.go       # Scrollable table component
│       ├── expander.go    # Click-to-expand component (for log events)
│       └── form.go        # Huh form wrapper for update operations
│
├── tests/
│   └── mocks/
│       ├── mock_server.go        # Generic mock HTTP server
│       └── auth0/                # Auth0-specific mock handlers by module
│           ├── users.go
│           ├── clients.go
│           ├── roles.go
│           ├── connections.go
│           └── logs.go
│
└── docs/
    ├── ARCHITECTURE.md    # This file
    ├── MODULES.md         # Module-specific details and patterns
    ├── OPENAPI_USAGE.md   # How to use the Auth0 OpenAPI spec
    └── PROJECT.md         # Current project status, decisions, open questions
```

---

## Layer Descriptions

### `cmd/`

CLI entry point using Charm's CLI framework (or cobra). Handles:
- `a0hero configure` — launches Huh form to collect `name`, `client_id`, `client_secret`, writes to `config/<name>.yaml`
- `a0hero run` — starts the TUI
- `a0hero export-logs` — export logs to JSON from CLI (TUI also supports this)

### `client/`

The Auth0 API client package. This is the only layer that knows about Auth0's HTTP endpoints. It is intentionally **importable without the TUI**, so other Go code can use it directly.

- `config.go` — loads `config/*.yaml`, supports env var override (`AUTH0_CLIENT_ID`, `AUTH0_CLIENT_SECRET`)
- `auth.go` — authentication via client credentials, token caching
- `client.go` — wraps `http.Client`, provides typed methods for each module
- `errors.go` — wraps Auth0 API error responses into typed Go errors

### `models/`

Shared types used across modules. Contains:
- Pagination types (`Page`, `Pagination`)
- Common response wrappers
- `Module` interface (if we define one for the module registry)

### `modules/`

One subdirectory per Auth0 resource. Each is self-contained:

- `client.go` — API calls specific to this module (uses the shared `client/` package)
- `model.go` — types specific to this module (some may be auto-generated from OpenAPI)
- `view.go` — Bubble Tea view for this module (or stub for later)

All modules are independently testable via the mock handlers in `tests/mocks/auth0/`.

### `tui/`

The Bubble Tea application:

- `app.go` — root model, holds the current module view, handles tenant context
- `main_menu.go` — landing view with tenant selector and module navigation
- `tenant.go` — tenant switching logic (resets all state on switch)
- `views/` — per-module views (scrollable tables, expandable rows, Huh forms for editing)
- `components/` — reusable UI components (table, expander, form)

The TUI **imports** modules but modules **do not import** the TUI. This is the key separation.

---

## Config Management

### Config Directory

```
config/
├── dev.yaml
├── tst.yaml
└── prod.yaml
```

### Config File Format

```yaml
name: dev
domain: dev-tenant.auth0.com
client_id: YOUR_CLIENT_ID
client_secret: YOUR_CLIENT_SECRET
```

### Config Loading Priority

1. Environment variables `AUTH0_CLIENT_ID` + `AUTH0_CLIENT_SECRET` (takes precedence)
2. Config file in `config/` directory matching the currently selected tenant

### Configure Flow

**First run:** If no `config/` files exist, the app launches the configure wizard automatically.

**On demand:** Running `a0hero configure` (or selecting from main menu) launches a Huh form:

1. Select or create a config file (dev/tst/prod or new)
2. Enter `name` (friendly label)
3. Enter `client_id`
4. Enter `client_secret`
5. Optionally: enter tenant domain (e.g. `dev-tenant.auth0.com`)
6. Test connection by attempting a lightweight API call (e.g. `/api/v2/clients`)
7. On success: write file and return to main menu
8. On failure: show error and let user retry

---

## Module Pattern

Each module follows this pattern. Take `modules/clients/` as an example:

```go
// client.go
func (c *Client) List(ctx context.Context, page int) (*ClientsResponse, error)
func (c *Client) Get(ctx context.Context, clientID string) (*ClientResponse, error)
func (c *Client) Create(ctx context.Context, body *CreateClientRequest) (*ClientResponse, error)
func (c *Client) Update(ctx context.Context, clientID string, body *UpdateClientRequest) (*ClientResponse, error)
func (c *Client) UpdateRedirectURIs(ctx context.Context, clientID string, uris []string) error
func (c *Client) Delete(ctx context.Context, clientID string) error
```

All API responses are defined as Go structs (manually or generated from OpenAPI). All errors are wrapped with context about which call failed.

---

## TUI Views

### Main Menu

The landing screen. Shows:
- Current tenant name and domain
- Module entry points: Users, Clients, Roles, Connections, Logs
- "Change Tenant" option (shows list of available config files)
- Configure option

### Module Views (Users, Clients, Roles, Connections)

- Scrollable table listing records with key columns
- Enter or click to open detail/edit view
- Huh form for editing fields (pre-filled with current values)
- Submit sends update to Auth0 API via module's client

### Logs View

- Scrollable list of log events (one-liner: timestamp, event type, description, user)
- Click/Enter to expand — shows full JSON object of the log event
- Date filter to narrow results (start date, end date)
- Export button — exports current filtered view to JSON file

### Tenant Switching

From the main menu, selecting "Change Tenant" shows a list of available config files. Selecting one:
- Resets all current view state (no carry-over)
- Loads the new config
- Returns to main menu showing the new tenant context

---

## OpenAPI-Driven Development

### The Spec

Auth0 Management API OpenAPI spec:
`https://auth0.com/docs/api/management/v2/openapi.yaml`

### How It's Used

1. **Model generation:** Use `oapi-codegen` to generate Go types from the spec
2. **Mock servers:** Generate mock HTTP handlers from the spec for testing
3. **Test coverage:** Tests use mock handlers that return spec-compliant responses
4. **Sync:** When Auth0 updates the spec, re-run codegen and update tests

### Workflow

```
1. Fetch latest spec
2. oapi-codegen -generate types -o models/generated.go
3. oapi-codegen -generate chi-server -o tests/mocks/auth0/server.go
4. Write tests against mock server
5. Implement module client against real Auth0 API
6. Tests pass → module done
```

See `docs/OPENAPI_USAGE.md` for the full workflow.

---

## Logging

All operations write structured logs. The TUI has a built-in log viewer (`modules/logs/`) that:
- Polls `/api/v2/logs` at a configurable interval (default: 30 seconds)
- Displays one-liner events: `[timestamp] [event_type] [description] [user]`
- Click to expand full JSON
- Export to JSON file with timestamp range in filename

### Log Levels

Use a structured logger (e.g. `log/slog` or `zerolog`). Fields should include:
- `ts` — ISO 8601 timestamp
- `level` — info, warn, error
- `tenant` — which tenant the operation acted on
- `module` — which module (users, clients, etc.)
- `action` — what happened (list, create, update, delete)
- `target` — resource ID if applicable
- `status` — success or error
- `error` — error message if failed

---

## Auth Flow

1. App starts → load config for currently selected tenant (or env vars)
2. Use `client_id` + `client_secret` to call Auth0's `oauth/token` endpoint (client credentials grant)
3. Cache the access token (it expires in 24 hours)
4. On token expiry, re-authenticate automatically
5. All API calls attach the token as `Authorization: Bearer <token>`

No device flow is needed since the client secret is stored in the config file. This is acceptable for local developer tooling.

---

## Module Independence — No Exceptions

**The `modules/` directory must never import the `tui/` package.**

This is the single most important architectural constraint in this project. It is not a guideline — it is a rule.

Why it matters:
- The `client/` package must be importable in other Go programs without pulling in Bubble Tea
- Modules must be testable without instantiating the TUI
- Future alternative interfaces (e.g. a CLI-only mode) must be able to import `modules/` directly
- It enforces honest layering: the TUI is a consumer of the domain, not part of it

If you find yourself wanting to import `tui/` from a module, the design is wrong. Go back to the module definition and find a better way to achieve what you need without crossing that boundary.

```
modules/  ──imports──►  client/
tui/      ──imports──►  modules/ + client/
cmd/      ──imports──►  client/ + modules/ + tui/
```

The only upward dependency is `tui/` consuming `modules/`. Nothing flows the other way.

---

## Dependencies

```go
// go.mod (expected)
module a0hero

go 1.24

require (
    github.com/charmbracelet/bubbletea v1.x    // TUI framework
    github.com/charmbracelet/huh v1.x          // Forms
    github.com/charmbracelet/lipgloss v1.x     // Styling
    github.com/charmbracelet/x/exp/cli         // Optional: for CLI helpers
    github.com/auth0/auth0-go v1.x             // Auth0 SDK (or use direct HTTP)
    github.com/oapi-codegen/oapi-codegen/v2    // OpenAPI code generation
    github.com/spf13/cobra                     // CLI commands
    gopkg.in/yaml.v3                           // Config file parsing
    github.com/stretchr/testify                // Testing
)
```