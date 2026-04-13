# A0Hero — Modules

## Overview

Each Auth0 resource is represented as a self-contained module under `modules/`. Every module follows the same structure and pattern, making them independently testable and easy to add.

---

## Module Structure

Every module has the same set of files:

```
modules/<name>/
├── client.go    # API calls (uses shared client/)
├── model.go     # Type definitions for this resource
└── view.go      # Bubble Tea view (or stub, may move to tui/views/)
```

A module may also have:
- `test.go` — integration tests using mock handlers
- `fixtures.go` — test fixtures (JSON bytes for mock responses)

---

## Module Interface Pattern

Each module's `client.go` exposes a consistent set of operations:

```go
type ModuleClient struct {
    httpClient *HTTPClient  // shared Auth0 HTTP client
}

func (m *ModuleClient) List(ctx context.Context, params ListParams) (*ListResponse, error)
func (m *ModuleClient) Get(ctx context.Context, id string) (*ItemResponse, error)
func (m *ModuleClient) Create(ctx context.Context, body *CreateRequest) (*ItemResponse, error)
func (m *ModuleClient) Update(ctx context.Context, id string, body *UpdateRequest) (*ItemResponse, error)
func (m *ModuleClient) Delete(ctx context.Context, id string) error
```

All methods return `context.Context` for cancellation. All errors wrap with the module name and action.

---

## Users Module

**Resource:** `/api/v2/users`

### Operations

| Method | Auth0 Endpoint | Description |
|--------|---------------|-------------|
| `List` | `GET /api/v2/users` | List users (paginated, filterable by email, name, etc.) |
| `Get` | `GET /api/v2/users/{id}` | Get a single user by Auth0 user ID |
| `Search` | `GET /api/v2/users?q=...` | Search users by query (email, name, etc.) |
| `Create` | `POST /api/v2/users` | Create a new user |
| `Update` | `PATCH /api/v2/users/{id}` | Update user fields |
| `Delete` | `DELETE /api/v2/users/{id}` | Delete a user |

### Model Fields (key subset)

```go
type User struct {
    ID          string            `json:"user_id,omitempty"`
    Email       string            `json:"email"`
    Name        string            `json:"name"`
    Picture     string            `json:"picture,omitempty"`
    CreatedAt   time.Time         `json:"created_at,omitempty"`
    UpdatedAt   time.Time         `json:"updated_at,omitempty"`
    LastLogin   time.Time         `json:"last_login,omitempty"`
    EmailVerified bool            `json:"email_verified"`
    AppMetadata map[string]any    `json:"app_metadata,omitempty"`
    UserMetadata map[string]any   `json:"user_metadata,omitempty"`
}
```

### Search/Filter Notes

Auth0 uses Lucene query syntax for `/users?q=`. Common patterns:
- `email:"john@email.nl"` — exact email match
- `name:"Jan"` — name contains
- `app_metadata.onas_account_id:"12345"` — metadata search

Use the `search` method rather than `list` for user lookup by email.

---

## Clients Module

**Resource:** `/api/v2/clients`

### Operations

| Method | Auth0 Endpoint | Description |
|--------|---------------|-------------|
| `List` | `GET /api/v2/clients` | List all applications/clients |
| `Get` | `GET /api/v2/clients/{id}` | Get a single client |
| `Create` | `POST /api/v2/clients` | Create a new application |
| `Update` | `PATCH /api/v2/clients/{id}` | Update client fields |
| `UpdateRedirectURIs` | `PATCH /api/v2/clients/{id}` | Specifically update `redirect_uris` |
| `Delete` | `DELETE /api/v2/clients/{id}` | Delete a client |

### Model Fields (key subset)

```go
type Client struct {
    ClientID     string   `json:"client_id,omitempty"`
    Name         string   `json:"name"`
    AppType      string   `json:"app_type,omitempty"`  // spa, regular_web, non_interactive, etc.
    callbacks    []string `json:"callbacks,omitempty"`
    redirect_uris []string `json:"redirect_uris,omitempty"`
    origins      []string `json:"origins,omitempty"`
    grantTypes   []string `json:"grant_types,omitempty"`
    LogoURI      string   `json:"logo_uri,omitempty"`
    Description  string   `json:"description,omitempty"`
}
```

### Updating Redirect URIs

The `UpdateRedirectURIs` method is a specialized update that only sends the `redirect_uris` field. This avoids sending the full client object.

```go
func (c *Client) UpdateRedirectURIs(ctx context.Context, clientID string, uris []string) error {
    body := map[string][]string{"redirect_uris": uris}
    return c.Patch(ctx, "/api/v2/clients/"+clientID, body)
}
```

---

## Roles Module

**Resource:** `/api/v2/roles`

### Operations

| Method | Auth0 Endpoint | Description |
|--------|---------------|-------------|
| `List` | `GET /api/v2/roles` | List all roles |
| `Get` | `GET /api/v2/roles/{id}` | Get a single role |
| `Create` | `POST /api/v2/roles` | Create a new role |
| `Update` | `PATCH /api/v2/roles/{id}` | Update role (name, description) |
| `Delete` | `DELETE /api/v2/roles/{id}` | Delete a role |
| `AssignUsers` | `POST /api/v2/roles/{id}/users` | Assign role to users |
| `RemoveUsers` | `DELETE /api/v2/roles/{id}/users` | Remove role from users |
| `GetPermissions` | `GET /api/v2/roles/{id}/permissions` | List permissions for a role |
| `AddPermissions` | `POST /api/v2/roles/{id}/permissions` | Add permissions to a role |

### Model Fields

```go
type Role struct {
    ID          string `json:"id,omitempty"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}
```

---

## Connections Module

**Resource:** `/api/v2/connections`

### Operations

| Method | Auth0 Endpoint | Description |
|--------|---------------|-------------|
| `List` | `GET /api/v2/connections` | List all connections |
| `Get` | `GET /api/v2/connections/{id}` | Get a single connection |
| `Create` | `POST /api/v2/connections` | Create a new connection |
| `Update` | `PATCH /api/v2/connections/{id}` | Update connection settings |
| `Delete` | `DELETE /api/v2/connections/{id}` | Delete a connection |

### Model Fields

```go
type Connection struct {
    ID                string   `json:"id,omitempty"`
    Name              string   `json:"name"`
    Strategy          string   `json:"strategy"`   // auth0, google-oauth2, etc.
    Options           map[string]any `json:"options,omitempty"`
    EnabledClients    []string `json:"enabled_clients,omitempty"`
}
```

### Use Cases

- View federated identity providers (e.g. Digidentity for eHerkenning)
- Enable/disable connections for an application
- Update OIDC provider settings

---

## Logs Module

**Resource:** `/api/v2/logs`

### Operations

| Method | Auth0 Endpoint | Description |
|--------|---------------|-------------|
| `List` | `GET /api/v2/logs` | List log events (paginated, filterable) |
| `Get` | `GET /api/v2/logs/{log_id}` | Get a single log event |

### Query Parameters (for List)

| Param | Description |
|-------|-------------|
| `from` | Log ID (cursor) to start from |
| `take` | Number of entries (default 50, max 100) |
| `include_totals` | Include total count in response |

Auth0 log events are not filterable by date directly — use `take` and cursor-based pagination to page through results. The log viewer maintains a timestamp of the oldest shown event so it can stop or warn when reaching old events.

### Model Fields

```go
type LogEvent struct {
    ID       string `json:"log_id"`
    Date     string `json:"date"`
    Type     string `json:"type"`         // e.g. "felo", "fai", "slo"
    IP       string `json:"ip,omitempty"`
    UserID   string `json:"user_id,omitempty"`
    UserName string `json:"user_name,omitempty"`
    Details  map[string]any `json:"details,omitempty"`
    Data     map[string]any `json:"data,omitempty"`  // Full event payload
}

type LogListResponse struct {
    Logs    []LogEvent `json:"logs"`
    Total   int        `json:"total,omitempty"`
}
```

### Log Type Codes (common)

| Type | Description |
|------|-------------|
| `felo` | Failed login (incorrect password or MFA failure) |
| `fai` | Failed login attempt (invalid credentials) |
| `slo` | Successful logout |
| `suca` | Successful login |
| `pacu` | Password change |
| `fc` | Failed by connector |
| `w` | Warning |

### One-Liner Display Format

```
[2026-04-13 14:23:01] [felo] Failed login — john@email.nl from 192.168.1.1
```

### Export Format

JSON file named: `logs-<from_date>-to-<to_date>.json`

```json
[
  {
    "log_id": "900000000000000000000000000000000000000000000",
    "date": "2026-04-13T14:23:01.000Z",
    "type": "felo",
    "ip": "192.168.1.1",
    "user_id": "auth0|abc123",
    "user_name": "john@email.nl",
    "details": { ... },
    "data": { ... }
  }
]
```

---

## Adding a New Module

To add a new module (e.g. `actions`):

1. Create `modules/actions/`
2. Add `client.go` with the full set of operations following the pattern
3. Add `model.go` with type definitions
4. Add `view.go` (or stub and put the view in `tui/views/actions.go`)
5. Add mock handlers in `tests/mocks/auth0/actions.go`
6. Add tests that fail against the mock
7. Implement until tests pass
8. Add entry point in `tui/main_menu.go`

No refactoring of existing modules is needed — the module registry is additive only.

---

## Shared Patterns

### Pagination

List operations return a `ListResponse` with a common shape:

```go
type ListResponse[T any] struct {
    Items []T
    Total int
    Next  string  // cursor for next page
}
```

The TUI passes this to the table component for rendering.

### Error Wrapping

All module errors are wrapped with module name and operation:

```go
return nil, fmt.Errorf("modules/clients: List: %w", err)
```

### Config Access in Modules

Modules receive the `client.HTTPClient` at construction time. They do not access config directly — the config is loaded once in `client.New()` and the authenticated client is passed to modules.

```go
// Construction
client, _ := client.New(ctx, "config/dev.yaml")
usersMod := users.New(client)
clientsMod := clients.New(client)
```

---

## TUI View Pattern

Each module's view in `tui/views/` follows the same pattern:

```go
type <Module>Model struct {
    client *module.Client
    items  []module.Item
    page   int
    loading bool
    error  error
}

func (m <Module>Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m <Module>Model) View() string { ... }
```

**Modules do not own TUI views.** The `modules/<name>/view.go` file is a stub or placeholder. Actual views live in `tui/views/`. This keeps the module boundary clean and prevents accidental TUI imports from bleeding into the domain layer.

Shared components used in `tui/components/`:
- `components.Table` — scrollable, paginated table
- `components.Expander` — click-to-expand row
- `components.Form` — Huh form with pre-filled values for update operations