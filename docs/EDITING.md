# Edit Mode — Design Specification

**Date:** 2026-04-14  
**Status:** Approved

---

## Overview

Edit mode allows in-place editing of Auth0 entities (Clients, Users, Roles, Connections) via a modal overlay. Changes are validated locally before submitting to the API, and a JSON audit log is written on successful save.

---

## User Interaction Flow

```
[List View] → press [e] → [Overlay: View Mode]
                                 ↓
                         press [e] again or [Edit]
                                 ↓
                        [Overlay: Edit Mode]
                                 ↓
                    make changes (dirty = true)
                                 ↓
                    press [Ctrl+S] or [Save]
                                 ↓
                    ┌────────────────────────┐
                    ↓                        ↓
              validation OK           validation FAIL
                    ↓                        ↓
              API PATCH request      show error popup
                    ↓                        ↓
                    └──────────┬───────────┘
                               ↓
                    ┌────────────────────────┐
                    ↓                        ↓
              2xx response            4xx/5xx response
                    ↓                        ↓
              write history JSON      show error popup
              return to view          stay in edit mode
                    ↓
              show success toast
```

---

## Key Bindings

| Key | Action |
|-----|--------|
| `e` | Enter edit mode (from view) |
| `Escape` | If dirty → prompt "Discard changes?" (Yes/No/Cancel) |
| `Ctrl+S` | Save changes |
| `Ctrl+Z` | Undo last field change (while editing) |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `q` | Close overlay (with dirty check) |

---

## Field Types

```
┌─────────────────────────────────────────────────────────────┐
│ Type          │ UI Component                                │
├───────────────┼─────────────────────────────────────────────┤
│ text          │ Single-line input                          │
│ textarea      │ Multi-line text area                       │
│ url           │ Single URL input with validation            │
│ tagArray      │ Chip/tag input (URLs, strings)              │
│ bool          │ Toggle switch (on/off)                     │
│ select        │ Dropdown with options                       │
│ password      │ Masked input with reveal toggle            │
│ datetime      │ Date/time picker                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Tag/Array Editing UI

```
┌─ Allowed Callback URLs ─────────────────────────────────────┐
│                                                              │
│  [https://google.com ×]  [http://localhost:3000 ×]         │
│                                                              │
│  [+ Add URL: _________________________________ ]            │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

- **Add item:** Type value + press `Enter`
- **Remove item:** Click `×` on tag or select tag + `Backspace`
- **Edit item:** Click on tag to enter inline edit mode

---

## Validation

### URL Validation
1. Must be valid URI format (`url.ParseRequestURI`)
2. Must have scheme (`http://` or `https://`)
3. No duplicate values within the same field

### Duplicate Detection
Arrays are checked for duplicates before save:
```
Error: "callback_urls" contains duplicate: "https://example.com/callback"
```

### Error Popup
```
┌─────────────────────────────────────────────────┐
│  ✗ Validation Failed                           │
├─────────────────────────────────────────────────┤
│                                                 │
│  • "https://bad-url" is not a valid URL        │
│  • "https://duplicate.com" is a duplicate     │
│                                                 │
│                              [ OK ]             │
└─────────────────────────────────────────────────┘
```

---

## Fetching Current State

Before entering edit mode:
1. Fetch full entity from API: `GET /api/v2/clients/{id}`
2. Populate form with current values
3. Store original snapshot for dirty detection

This ensures:
- All fields are available (list view may be truncated)
- Validation can compare against server state
- Change history captures accurate old values

---

## Change History JSON

### Location
```
~/.a0hero/history/<entity-type>/<entity-id>/<timestamp>.json
```

Example:
```
~/.a0hero/history/client/abc123xyz/2026-04-14T103045.json
```

### Structure

```json
{
  "version": 1,
  "session_id": "2026-04-14T10:30:00Z-editor",
  "entity_type": "client",
  "entity_id": "abc123xyz",
  "started_at": "2026-04-14T10:30:00Z",
  "ended_at": "2026-04-14T10:35:00Z",
  "user_agent": "a0hero/v0.0.3 (darwin/arm64)",
  "changes": [
    {
      "timestamp": "2026-04-14T10:31:15Z",
      "field": "name",
      "old_value": "My App",
      "new_value": "My Updated App"
    },
    {
      "timestamp": "2026-04-14T10:32:00Z",
      "field": "callbacks",
      "old_value": ["https://old.com/callback"],
      "new_value": ["https://old.com/callback", "https://new.com/callback"]
    },
    {
      "timestamp": "2026-04-14T10:33:30Z",
      "field": "web_origins",
      "old_value": [],
      "new_value": ["https://myapp.com"]
    }
  ],
  "final_state": {
    "name": "My Updated App",
    "callbacks": ["https://old.com/callback", "https://new.com/callback"],
    "web_origins": ["https://myapp.com"],
    ...
  },
  "status": "saved",
  "api_response": {
    "status_code": 200,
    "duration_ms": 234
  }
}
```

### Redacted Fields
The following fields are redacted from history:
- `client_secret`
- `encryption_key`
- Any field matching `*password*` (case-insensitive)

### Status Values
- `saved` — Successfully saved to API
- `cancelled` — User abandoned changes
- `failed` — API returned error

---

## Extensibility Pattern

### Module Definition

Each module (`modules/<name>/`) defines editing capabilities:

```go
// modules/clients/fields.go

package clients

type FieldType int

const (
    fieldText FieldType = iota
    fieldTextarea
    fieldURL
    fieldTagArray
    fieldBool
    fieldSelect
    fieldPassword
)

type FieldDef struct {
    Key       string
    Label     string
    Type      FieldType
    Required  bool
    ReadOnly  bool
    Options   []string      // for select
    Validate  func(val interface{}) error
}

// Fields returns the editable fields for a Client
func Fields() []FieldDef {
    return []FieldDef{
        {Key: "name",        Label: "Name",              Type: fieldText},
        {Key: "description", Label: "Description",       Type: fieldTextarea},
        {Key: "app_type",   Label: "Application Type", Type: fieldSelect,
            Options: []string{"spa", "regular_web", "native", "non_interactive", "machine"}},
        {Key: "callbacks",   Label: "Allowed Callback URLs", Type: fieldTagArray,
            Validate: ValidateURLs},
        {Key: "web_origins", Label: "Web Origins",       Type: fieldTagArray,
            Validate: ValidateURLs},
        {Key: "logout_urls", Label: "Logout URLs",        Type: fieldTagArray,
            Validate: ValidateURLs},
        {Key: "login_uri",   Label: "Custom Login Page URL", Type: fieldURL,
            Validate: ValidateURL},
        // ...
    }
}

// BuildUpdatePayload returns only changed fields for PATCH
func BuildUpdatePayload(original, updated map[string]interface{}, fields []FieldDef) map[string]interface{} {
    // Only include fields that changed
}
```

### Validation Functions

```go
// modules/clients/validation.go

func ValidateURL(val interface{}) error {
    urls := val.([]string)
    seen := make(map[string]bool)
    for _, u := range urls {
        if _, err := url.ParseRequestURI(u); err != nil {
            return fmt.Errorf("%q is not a valid URL", u)
        }
        if seen[u] {
            return fmt.Errorf("%q is a duplicate", u)
        }
        seen[u] = true
    }
    return nil
}
```

---

## TUI Components

### EditableField Interface
```go
type EditableField interface {
    Render(label string, value interface{}) string
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    Value() interface{}
    SetValue(interface{})
    IsDirty() bool
    Reset()
}
```

### Implementation Order
1. ✅ Core types and field definitions (`FieldType`, `FieldDef`)
2. ✅ Validation utilities (`ValidateURLs`, `ValidateNoDuplicates`)
3. ✅ History writer (`HistoryWriter`)
4. ✅ TagArray component (Huh-based)
5. ✅ EditOverlay view (`tui/views/edit_overlay.go`)
6. ✅ Keyboard handling integration
7. ✅ Error popup component
8. ✅ Dirty state tracking
9. ✅ Ctrl+S / Save flow

---

## API Integration

### PATCH Request
```go
func (c *Client) Update(ctx context.Context, id string, payload map[string]interface{}) (*Client, error) {
    req, err := c.client.NewRequest("PATCH", "clients/"+id, payload)
    if err != nil {
        return nil, err
    }

    var result Client
    resp, err := c.client.Do(req, &result)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode >= 400 {
        return nil, parseAPIError(resp)
    }

    return &result, nil
}
```

---

### "More" Section

Replaces "Configure" as a catch-all for utilities:

```
┌─ More ─────────────────────────────────────────────────┐
│                                                        │
│  Configure                                             │
│  ─────────────────────────────                         │
│                                                        │
│  ➤ Clients History                                     │
│    Users History                                        │
│    Roles History                                        │
│    ...                                                  │
│                                                        │
└────────────────────────────────────────────────────────┘
```

History viewer shows past changes:

```
┌─ History: client/abc123 ──────────────────────────────┐
│                                                        │
│  Apr 14, 2026 10:30:45 — SAVED                         │
│  Apr 12, 2026 14:22:10 — SAVED                         │
│  Apr 10, 2026 09:15:33 — CANCELLED                     │
│                                                        │
│  [Select a history entry to view details]              │
│                                                        │
└────────────────────────────────────────────────────────┘

┌─ Change Details ──────────────────────────────────────┐
│                                                        │
│  name: "My App" → "My Updated App"                    │
│  callbacks: +1 item                                    │
│    - "https://new.com/callback" added                 │
│  web_origins: +1 item                                  │
│    - "https://myapp.com" added                        │
│                                                        │
│  Status: SAVED  •  Apr 14, 2026 10:30:45               │
│                                                        │
└────────────────────────────────────────────────────────┘
```

```
a0hero/
├── docs/
│   └── EDITING.md                    ← This spec
├── modules/
│   └── clients/
│       ├── client.go                 ← Existing
│       ├── fields.go                 ← Field definitions
│       ├── validation.go             ← Validators
│       └── history.go                ← Change history writer
├── tui/
│   ├── app.go                        ← Keyboard handling
│   └── views/
│       ├── detail_overlay.go         ← View mode (existing?)
│       ├── edit_overlay.go            ← Edit mode
│       └── components/
│           ├── tag_input.go          ← Tag/chip component
│           ├── error_popup.go        ← Error modal
│           └── field_renderers.go    ← Per-type renderers
```

---

## Open Questions

- [x] Should undo (`Ctrl+Z`) be limited to a stack depth, or unlimited? **→ Unlimited until no more changes**
- [x] Do we need a "View History" command to display past changes? **→ Yes, rename "Configure" to "More" and add History viewer**
- [x] Should history files be gzipped to save space? **→ No, plain JSON**
- [x] Implementation approach? **→ Scaffold extensible base, then Clients module**

---

## Implementation Status

### Completed ✓
- [x] `modules/edit/field_types.go` - Field types, validators, ValidationError
- [x] `modules/edit/history.go` - HistoryWriter, EditSession, Change tracking
- [x] `modules/edit/entity.go` - EntityService interface, FieldHelper
- [x] `tui/views/components/tag_input.go` - Tag/chip input component
- [x] `tui/views/components/error_popup.go` - Error modal
- [x] `tui/views/components/confirm_dialog.go` - Confirmation dialog
- [x] `tui/views/edit_overlay.go` - Edit overlay view

### TODO
- [ ] `modules/clients/fields.go` - Client field definitions
- [ ] `modules/clients/validation.go` - Client-specific validators
- [ ] Integrate edit overlay into main app (`tui/app.go`)
- [ ] Rename "Configure" to "More" section
- [ ] History viewer view

### Client Fields (to implement)

```go
// modules/clients/fields.go
var ClientFields = []edit.FieldDef{
    // Section: Application Properties
    {Key: "name", Label: "Name", Type: edit.FieldText, Required: true},
    {Key: "description", Label: "Description", Type: edit.FieldTextarea},
    {Key: "app_type", Label: "Application Type", Type: edit.FieldSelect, 
     Options: []string{"spa", "regular_web", "native", "non_interactive", "machine"}},
    {Key: "logo_uri", Label: "Application Logo URL", Type: edit.FieldURL},
    {Key: "owners", Label: "Application Owners", Type: edit.FieldTagArray},
    
    // Section: Login URLs
    {Key: "login_uri", Label: "Custom Login Page URL", Type: edit.FieldURL},
    {Key: "login_origin", Label: "Custom Login Origin", Type: edit.FieldURL},
    
    // Section: Security URLs  
    {Key: "callbacks", Label: "Allowed Callback URLs", Type: edit.FieldTagArray, 
     Validate: edit.ValidateURLArray},
    {Key: "logout_urls", Label: "Allowed Logout URLs", Type: edit.FieldTagArray,
     Validate: edit.ValidateURLArray},
    {Key: "web_origins", Label: "Web Origins", Type: edit.FieldTagArray,
     Validate: edit.ValidateURLArray},
    
    // Read-only fields
    {Key: "client_id", Label: "Client ID", Type: edit.FieldText, ReadOnly: true},
}
```
