# A0Hero Architecture

> Onboarding reference for bringing up the project on a new machine.

## Quick Start

```bash
git clone https://github.com/samrocksc/a0hero.git
cd a0hero
go build ./cmd/a0hero
./a0hero --debug          # runs with debug logging to logs/
./a0hero --help           # all flags
```

Config lives in `~/.config/a0hero/<tenant>.yaml`:
```yaml
name: my-tenant
domain: my-tenant.auth0.com
client_id: XXXX
client_secret: YYYY
```

Auth0 credentials can also come from env vars: `AUTH0_CLIENT_ID`, `AUTH0_CLIENT_SECRET`, `AUTH0_DOMAIN`.

---

## Architecture Principles

1. **Strict layer separation** — `modules/` never imports `tui/`. Domain logic and presentation are independent.
2. **TDD always** — No production code without a failing test first.
3. **Config is king** — No hardcoded tenants or domains. Multi-tenant via config files.
4. **Extensible by design** — New modules follow the same pattern: client.go, fields.go, model.go.
5. **Meaningful logs** — All operations log structured JSON to `logs/`. Use `--debug` flag.

---

## Project Structure

```
a0hero/
├── cmd/a0hero/main.go        # CLI entry point (Cobra + Bubble Tea)
├── client/                    # Auth0 API client, auth, config
│   ├── auth.go               # Token management, refresh, timeout
│   ├── client.go             # HTTP client with auth transport
│   ├── config.go             # YAML config loading
│   └── errors.go             # API error types
├── modules/                   # Domain modules (NO tui imports)
│   ├── edit/                  # Edit framework (field types, sessions, history)
│   │   ├── field_types.go    # FieldType enum, FieldDef, validators
│   │   ├── entity.go         # EntityService interface (Fetch/Update)
│   │   └── history.go        # EditSession, Change tracking, history writer
│   ├── clients/              # Auth0 Clients module
│   │   ├── client.go         # Auth0Client (List/Fetch/Update + EntityService)
│   │   └── fields.go         # ClientFields definition (all editable fields)
│   ├── users/                # Users module
│   ├── roles/                # Roles module
│   ├── connections/           # Connections module
│   └── logs/                 # Logs module
├── models/common.go           # Shared types (moduleItem, Row() interface)
├── tui/                       # Bubble Tea UI layer
│   ├── app.go                # Root model — sections, key handling, caching, state
│   ├── components/            # Shared UI components
│   │   └── table.go           # Table renderer
│   └── views/                 # Overlay views
│       ├── edit_overlay.go    # Inline edit view (view/edit/field-input modes)
│       └── components/       # View-specific components (confirm, error, tag input)
├── logger/logger.go            # Structured JSON logger (--debug flag)
├── version/version.go         # Build version info (set via -ldflags)
└── docs/                      # Documentation
    ├── ARCHITECTURE.md        # This file
    └── EDITING.md             # Edit mode design spec
```

---

## Data Flow

```
┌──────────┐     ┌─────────────────┐     ┌──────────────────┐
│  Config   │────▶│  client.Auth    │────▶│  Auth0 Management │
│  (YAML)  │     │  (token refresh) │     │  API (REST)        │
└──────────┘     └─────────────────┘     └──────────────────┘
                        │                        │
                        ▼                        ▼
                 ┌─────────────┐        ┌─────────────────┐
                 │  App model   │        │  modules/*       │
                 │  (app.go)    │◀───────│  (domain logic) │
                 └──────┬──────┘        └─────────────────┘
                        │
                ┌───────┴───────┐
                │               │
          ┌─────▼─────┐  ┌─────▼──────┐
          │  Edit      │  │  Section    │
          │  Overlay   │  │  Views      │
          └────────────┘  └─────────────┘
```

1. **Config** loaded at startup → creates `client.Client` with auth transport
2. **App** manages section state, key routing, caching
3. **Modules** (`clients/`, `users/`, etc.) contain domain logic, `List()`/`Fetch()`/`Update()` methods
4. **EditOverlay** is a Bubble Tea sub-model that handles the edit lifecycle (fetch → view → edit → save)

---

## Key Bindings (TUI)

| Key | Context | Action |
|-----|---------|--------|
| `tab`/`l`/`→` | Any | Next section |
| `shift+tab`/`h`/`←` | Any | Previous section |
| `↑`/`k` | List | Move cursor up |
| `↓`/`j` | List | Move cursor down |
| `enter` | List | View detail overlay |
| `esc` | Detail/Edit | Close overlay / cancel |
| `e` | List | Open edit overlay |
| `q` | View mode | Quit |

### Edit Overlay Keys

| Key | Mode | Action |
|-----|------|--------|
| `e` | View | Enter edit mode |
| `esc` | View | Close overlay |
| `esc` | Edit | Cancel edit, return to view |
| `esc` | Field input | Confirm value, return to field nav |
| `enter` | Edit | Start typing into focused field |
| `enter` | Field input | Confirm value (text) / add tag, move to next |
| `ctrl+s` | Edit | Save all changes |
| `ctrl+z` | Edit/Field | Undo change |
| `↑`/`↓` | Edit | Navigate fields |

---

## Edit Flow

```
Press 'e' on item
    ↓
Fetch entity from API (10s timeout)
    ↓
Display in VIEW mode (label: value pairs)
    ↓
Press 'e' → EDIT mode (fields highlighted, enter to type)
    ↓
Press 'enter' on field → start typing
    ↓
Type value, press 'esc' → commit value, back to field nav
    ↓
Press 'ctrl+s' → save all changes via API PATCH
    ↓
Success: show "Saved!", return to view mode
Failure: show inline error
```

---

## Module Pattern

Each module follows this exact pattern:

```go
// modules/<name>/client.go
type Auth0Client struct { c *client.Client }

func New(c *client.Client) *Auth0Client { ... }
func (a *Auth0Client) List(ctx context.Context) ([]Type, error) { ... }
func (a *Auth0Client) Fetch(ctx context.Context, id string) (map[string]interface{}, error) { ... }
func (a *Auth0Client) Update(ctx context.Context, id string, payload map[string]interface{}) error { ... }
```

The `Fetch` and `Update` methods implement the `edit.EntityService` interface, making them usable by the edit overlay without any coupling to the TUI.

### Adding a new module

1. Create `modules/<name>/client.go` with `List`, `Fetch`, `Update`
2. Create `modules/<name>/fields.go` with field definitions
3. Add a section constant and fetch method in `tui/app.go`
4. Add columns via `Columns()` method
5. Wire the `e` key to open `EditOverlay` with the module's fields and service

---

## Caching

- Section data (users, clients, etc.) cached for 30 seconds
- Cache cleared on tenant switch and on save
- Cache bypassed on explicit refresh (future feature)

---

## History / Audit Log

On successful save:
```
~/.a0hero/history/<type>/<id>/<timestamp>.json
```

Contains: session_id, entity_type, entity_id, changes (field, old_value, new_value), final_state, status, API response.

Sensitive fields (`client_secret`, `*password*`) are redacted.

---

## Testing

```bash
go test ./...                    # all tests
go test ./modules/clients/...    # specific module
go test ./client/...             # auth and client tests
```

Tests use mock HTTP handlers against the Auth0 OpenAPI spec. See `docs/OPENAPI_USAGE.md` for details.

---

## Building & Releasing

```bash
make build            # build binary
make test             # run tests
make lint             # golangci-lint
make release          # cross-compile + tag

# Build with version info:
go build -ldflags "-X version.Version=v0.1.0 -X version.Commit=$(git rev-parse --short HEAD)" ./cmd/a0hero
```

Current version: **v0.1.0**

---

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Bubble Tea + Lip Gloss | Best Go TUI framework, composable, terminal-agnostic |
| Strict layer separation | Modules must be testable without TUI imports |
| YAML config per tenant | Multi-tenant support, no database needed |
| Inline edit (not modal) | Better UX — see context while editing |
| Field-level editing (not form) | More natural: enter to type, esc to confirm |
| History JSON on save | Audit trail without database |
| 10s API timeout + ctx cancellation | No hung requests, responsive UI |
| 30s section cache | Fast tab switching, stale-data-safe |