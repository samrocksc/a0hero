# A0Hero — Proposed Architecture

## Overview

Refactor A0Hero into a clean 3-tier architecture with separated concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                     PRESENTATION (TUI)                          │
│   Bubble Tea UI / Views / Components                            │
│   - Renders data and user interactions                          │
│   - Delegates business decisions to state machine              │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      BUSINESS LOGIC                              │
│   State Machine / Edit Sessions / Field Validation              │
│   - Orchestrates data operations                                │
│   - Manages edit state and change tracking                      │
│   - Validates field values against schemas                      │
└──────────────────────────┬────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                         DATA LAYER                               │
│   API Client Libraries                                           │
│   - Encapsulates all HTTP/REST communication                     │
│   - Handles authentication, token refresh, retries               │
│   - Returns typed domain models                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Issue: Boolean Field Handling

### Current Problem

`is_global` and `is_first_party` are sent as `"true"/"false"` strings or Go `bool` values, but Auth0's Management API expects specific formats for certain fields.

Example error:
```
Payload validation error: 'Additional properties not allowed: is_global'
```

### Proposed Solution

**Option A: Mark read-only fields at the API level**
- Fields like `is_global` should be excluded from PATCH payloads entirely
- The field is readable (GET returns it) but not writable (PATCH rejects it)

**Option B: Field type with API metadata**
```go
type FieldDef struct {
    Key       string
    Label     string
    Type      FieldType
    ReadOnly  bool           // Cannot be updated via PATCH
    Immutable bool           // Cannot be set even on create
    Required  bool
    // ... other fields
}
```

**Recommendation:** Option B with both `ReadOnly` and `Immutable` flags. `ReadOnly` means the field appears in forms but isn't saved. `Immutable` means it can only be set during creation.

---

## Data Layer: API Client Libraries

### Split Strategy

Current: API logic is scattered across `client/`, `modules/clients/`, `modules/users/`, etc.

Proposed: Two focused libraries:

```
lib/
├── auth0/                    # Auth0 Management API client
│   ├── client.go             # Base HTTP client, auth, retries
│   ├── clients.go            # /api/v2/clients endpoints
│   ├── connections.go        # /api/v2/connections endpoints
│   ├── roles.go              # /api/v2/roles endpoints
│   └── logs.go               # /api/v2/logs endpoints
│
└── auth0-users/              # Auth0 User Management API client
    └── users.go              # /api/v2/users endpoints
```

### Base Auth0 Client (`lib/auth0/client.go`)

```go
package auth0

type Client struct {
    baseURL    string
    httpClient *http.Client
    auth       *Authenticator  // Token management
}

func New(domain, clientID, clientSecret string) (*Client, error)
func (c *Client) Clients() *ClientsResource
func (c *Client) Connections() *ConnectionsResource
func (c *Client) Roles() *RolesResource
func (c *Client) Logs() *LogsResource
```

### Resource Pattern

Each resource is a self-contained struct that holds a reference to the base client:

```go
type ClientsResource struct {
    client *Client
}

func (r *ClientsResource) List(ctx context.Context) ([]Client, error)
func (r *ClientsResource) Get(ctx context.Context, id string) (*Client, error)
func (r *ClientsResource) Update(ctx context.Context, id string, changes map[string]any) (*Client, error)
func (r *ClientsResource) Create(ctx context.Context, req *CreateClientRequest) (*Client, error)
func (r *ClientsResource) Delete(ctx context.Context, id string) error
```

### Benefits

1. **Single responsibility** - each lib handles one API domain
2. **Shared auth** - base client handles token refresh for all resources
3. **Testable** - mock the base client for unit tests
4. **Versionable** - libs can be versioned independently if needed
5. **Discoverable** - clear entry points for each API resource

### Migration Path

1. Create `lib/auth0/` structure alongside existing `client/`
2. Move/copy API logic from `modules/*/client.go` into lib
3. Update `modules/*/client.go` to delegate to lib
4. Eventually deprecate `client/` package

---

## State Machine Integration

The state machine should be orchestration-only, delegating I/O to the data layer:

```
StateMachine
    │
    ├── ProcessEvent(EventSubmit)
    │
    ▼
StateMachine.TransitionTo(StateAPICall)
    │
    ▼
createAPICallCmd()
    │
    ├── Get entity service for section
    │
    ▼
entityService.Update(ctx, id, changes)
    │
    ▼
lib/auth0.Clients.Update(ctx, id, changes)  // <-- data layer
```

The state machine never makes HTTP calls directly. It uses injected `EntityService` implementations that delegate to the data layer.

---

## Open Questions

1. Should `lib/auth0/` be a separate Go module that can be used standalone?
2. How should we handle rate limiting from Auth0? (retry with backoff)
3. Should we cache token refresh failures and circuit-break?
4. What's the strategy for handling Auth0 API version changes?

---

## Priority

1. **Immediate** (fix current bugs): Mark `is_global` as read-only, fix boolean serialization
2. **Short-term**: Create `lib/auth0/` structure, refactor one module (e.g., clients)
3. **Medium-term**: Migrate all modules to use lib, remove old `client/` package