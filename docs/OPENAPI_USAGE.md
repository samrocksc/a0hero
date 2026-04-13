# A0Hero — OpenAPI Usage

## Overview

A0Hero uses the Auth0 Management API OpenAPI specification as the single source of truth for API models, behavior, and test mock generation. All TDD tests are driven by the spec.

---

## The Spec

Auth0 publishes the Management API v2 spec at:

```
https://auth0.com/docs/api/management/v2/openapi.yaml
```

Bookmark this URL. It's the anchor for the entire development process.

---

## Workflow

### Step 1 — Fetch the Spec

Store a local copy in the repo so it's versioned and available offline:

```bash
curl -s https://auth0.com/docs/api/management/v2/openapi.yaml -o tests/mocks/auth0/openapi.yaml
```

Commit the spec file. Update it when Auth0 ships a new version.

### Step 2 — Generate Go Types

Use `oapi-codegen` to generate struct types from the spec:

```bash
oapi-codegen \
  -generate types \
  -package auth0 \
  -o models/generated.go \
  tests/mocks/auth0/openapi.yaml
```

This generates all request/response types for every endpoint. Keep generated types in `models/generated.go` and add hand-written overrides in `models/` alongside it.

### Step 3 — Generate Mock Server

Generate a mock HTTP server from the spec:

```bash
oapi-codegen \
  -generate chi-server \
  -package mockauth0 \
  -o tests/mocks/auth0/server.go \
  tests/mocks/auth0/openapi.yaml
```

The mock server implements the spec's endpoints and returns spec-compliant responses. Use this as the foundation for test handlers.

### Step 4 — Write Tests Against the Mock

Each module has its own test file. Tests use the mock server to verify:

- The client correctly parses spec-compliant responses
- The client correctly handles error responses (4xx, 5xx)
- The client correctly serializes request bodies

```go
func TestUsersClient_List(t *testing.T) {
    // Setup mock server with spec-compliant response
    ts := mockauth0.NewTestServer(t, "tests/mocks/auth0/openapi.yaml")
    
    // Register handler for GET /api/v2/users
    ts.Handle("/api/v2/users", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, UsersListResponse{
            Users: []User{{UserID: "auth0|123", Email: "test@example.com"}},
            Total: 1,
        })
    })
    
    // Test client against mock
    client := NewClient(ts.URL(), "test-token")
    resp, err := client.Users.List(context.Background(), nil)
    
    require.NoError(t, err)
    require.Len(t, resp.Users, 1)
}
```

### Step 5 — Implement the Module

With tests written and failing, implement the module client to make tests pass.

### Step 6 — Verify Against Real API

Once tests are green, smoke-test against the actual Auth0 API (dev tenant) to confirm behavior matches. This is a manual step — not automated in CI.

### Step 7 — Sync When Spec Changes

When Auth0 updates the spec:
1. Pull new spec: `curl ... -o tests/mocks/auth0/openapi.yaml`
2. Re-run codegen: `oapi-codegen ...`
3. Review diff — new fields may need new tests
4. Update existing tests if response shapes changed

---

## oapi-codegen Quick Reference

```bash
# Generate types
oapi-codegen -generate types -package auth0 -o models/generated.go <spec.yaml>

# Generate mock server (chi)
oapi-codegen -generate chi-server -package mockauth0 -o tests/mocks/auth0/server.go <spec.yaml>

# Generate client (helpful but we write our own)
oapi-codegen -generate client -package auth0 -o client/generated.go <spec.yaml>
```

Run `oapi-codegen --help` for the full list of generators.

---

## Handling Auth in Tests

The mock server validates the `Authorization: Bearer <token>` header. In tests, use a dummy token:

```go
ts := mockauth0.NewTestServer(t, "tests/mocks/auth0/openapi.yaml", mockauth0.WithToken("test-token"))
```

For 401 responses, register an endpoint that returns 401 and verify the client propagates it correctly.

---

## Testing the Logs Module

The logs endpoint is unique — it returns very large JSON objects. In tests, use a representative subset of the log event shape:

```go
mockLogEvent := LogEvent{
    ID:       "900000000000000000000000000000000000000000000",
    Date:     "2026-04-13T14:23:01.000Z",
    Type:     "felo",
    IP:       "192.168.1.1",
    UserID:   "auth0|abc123",
    UserName: "test@example.com",
    Details:  map[string]any{"description": "Failed login"},
    Data:     map[string]any{"user_name": "test@example.com"},
}
```

Test pagination by having the mock return cursor-based `next` values.

---

## Test Fixtures

Store JSON fixtures for complex responses in `tests/fixtures/` as `.json` files. Load them in tests:

```go
fixture, _ := os.ReadFile("tests/fixtures/users_list_page1.json")
var resp UsersListResponse
json.Unmarshal(fixture, &resp)
```

This keeps test code clean and makes fixtures easy to update.

---

## Coverage Goals

| Module | Coverage Target |
|--------|----------------|
| clients | 100% — every endpoint, every field |
| users | 100% — every endpoint |
| roles | 100% — every endpoint |
| connections | 100% — list, get, create, update |
| logs | 100% — list (date filter, pagination), get |

---

## Common Issues

**Spec says field is `*string` but actual API returns `""`**
Auth0's spec sometimes uses pointers for optional fields. Our generated models may need manual adjustment to match actual behavior. If in doubt, test against dev tenant.

**Spec is out of date with actual API behavior**
If you discover a discrepancy, flag it in `docs/PROJECT.md` and work from what the actual API returns. The spec is authoritative until proven wrong.

**Mock server doesn't support all response variations**
Extend the mock server with custom handlers per-test rather than modifying the generated server. Custom handlers override generated ones.