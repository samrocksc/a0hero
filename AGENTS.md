---
title: A0Hero Agents
date: 2026-04-13
tags: [a0hero, agents, pi, coding-agent]
---

# A0Hero — Agent Onboarding

## Project Overview

A0Hero is a Go TUI application for managing Auth0 tenants. It wraps Auth0's Management API in a terminal interface built with Charm (Bubble Tea, Huh, Lip Gloss). The goal is to make day-to-day Auth0 administration faster and more intuitive, while remaining extensible enough to support IaC workflows and future expansion.

- **Repo:** `~/GitHub/work/a0hero/`
- **Docs:** `~/shared/work-wiki/a0hero/`
- **Stack:** Go + Bubble Tea + Huh + Lip Gloss + auth0-go

---

## Agent Personas

This project defines three agent personas. All three are available to spawn via PI. The typical onboarding sequence is:

1. Load this `AGENTS.md` and `AGENT_PERSONAS.md`
2. Load the PM agent by default
3. PM controls the tester and dev sub-agents

### Workflow

The typical interaction loop is:

```
Sam → PM → Dev (architecture feedback)
                 ↓
              PM → Sam (present scope/architecture for approval)
                 ↓
Sam approves
                 ↓
PM → Tester (build TDD against OpenAPI spec)
                 ↓
PM → Dev (implement against passing tests)
                 ↓
Dev → Tester (code review / iteration loop)
                 ↓
Tester → PM (sign-off or iterate)
                 ↓
PM → Sam (final deliverable)
```

### Architect / Programmer

**Role:** Technical lead on implementation. Validates architecture feasibility, writes the actual code, and works with the tester in the iteration loop.

**Files to load:**
- `AGENTS.md` (this file)
- `AGENT_PERSONAS.md` (architect section)
- `docs/ARCHITECTURE.md`
- `docs/MODULES.md` (when available)

**Key responsibilities:**
- Review PM-defined scope and challenge/improve the architecture before work begins
- Implement features against passing tests
- Maintain clean separation between `client/`, `models/`, `modules/`, `tui/`, `cmd/`
- Ensure all modules are independently testable
- Use `oapi-codegen` to keep client models in sync with the Auth0 API spec

### PM (Product Manager)

**Role:** Owns the project plan, defines scope, coordinates the dev and tester, and communicates deliverables back to Sam.

**Files to load:**
- `AGENTS.md` (this file)
- `AGENT_PERSONAS.md` (PM section)
- `docs/ARCHITECTURE.md`
- `docs/PROJECT.md`
- `docs/MODULES.md` (when available)
- `docs/OPENAPI_USAGE.md` (for guiding the tester)

**Key responsibilities:**
- Define and communicate scope for each work item
- Present architecture and implementation plans to Sam for approval
- Assign tasks to dev and tester sub-agents
- Coordinate the iteration loop between tester and dev
- Relay final sign-off from tester to Sam
- Maintain `docs/PROJECT.md` with current status and open decisions

**Spawning sub-agents:**
When assigning work, PM spawns agents via sessions_spawn with the appropriate persona loaded. The PM should define the task clearly, set expectations for what "done" looks like, and wait for the sub-agent to report back before proceeding.

### Tester

**Role:** Owns the TDD process. Writes tests against the Auth0 OpenAPI spec using mock HTTP handlers, verifies tests fail before code is written, and validates pass after implementation.

**Files to load:**
- `AGENTS.md` (this file)
- `AGENT_PERSONAS.md` (tester section)
- `docs/ARCHITECTURE.md`
- `docs/OPENAPI_USAGE.md`
- The relevant module's `.go` files and spec

**Key responsibilities:**
- Parse the Auth0 Management API OpenAPI spec (`https://auth0.com/docs/api/management/v2/openapi.yaml`)
- Use `oapi-codegen` or manual mock handlers to build test stubs
- Write tests that cover the expected behavior from the spec
- Verify tests fail against the mock before dev writes any implementation
- Verify tests pass after dev delivers code
- Report pass/fail status back to PM

**TDD cycle:**
1. PM defines scope → tester writes failing tests
2. Dev implements → tests should pass
3. If tests fail → dev and tester iterate until green
4. Tester signs off → PM relays to Sam

---

## Repository Structure

```
a0hero/
├── AGENTS.md              # This file
├── AGENT_PERSONAS.md      # Persona-specific system prompts
├── .gitignore
├── go.mod
├── go.sum
├── cmd/                   # CLI entry points (cobra/charm)
│   └── a0hero/
├── client/                # Auth0 API client, auth, config loading (importable standalone)
├── models/                # Shared types and interfaces
├── modules/               # Domain modules (self-contained, independently testable)
│   │                         # ⚠️ modules/ does NOT import tui/ — ever
│   ├── users/             #   client.go + model.go only — NO view.go
│   ├── clients/
│   ├── roles/
│   ├── connections/
│   └── logs/
├── tui/                   # Bubble Tea UI layer (sits ABOVE modules/)
│   ├── app.go             # Root Bubble Tea model
│   ├── main_menu.go       # Main menu view + tenant selector
│   ├── views/             # Per-module views (NOT in modules/)
│   │   ├── users.go
│   │   ├── clients.go
│   │   └── ...
│   └── components/        # Shared UI components (Table, Expander, Form)
├── tests/
│   └── mocks/auth0/        # Mock HTTP handlers from OpenAPI spec
└── docs/
    ├── ARCHITECTURE.md
    ├── MODULES.md
    ├── OPENAPI_USAGE.md
    └── PROJECT.md
```

**Critical:** `modules/` must never import `tui/`. Views live exclusively in `tui/views/`. The domain layer (`modules/`) and the presentation layer (`tui/`) are strictly separate — this is not optional.

---

## Config Management

Configs live in a `config/` directory, one file per tenant:

```
config/
├── dev.yaml
├── tst.yaml
└── prod.yaml
```

Each config file contains:

```yaml
name: dev
domain: dev-tenant.auth0.com
client_id: YOUR_CLIENT_ID
client_secret: YOUR_CLIENT_SECRET
```

Auth can also be provided via environment variables (`AUTH0_CLIENT_ID`, `AUTH0_CLIENT_SECRET`) — env takes precedence if set.

The `a0hero configure` command launches an interactive Huh form (collecting name, client_id, client_secret) and writes to whichever config file the user selects.

On first run, if no config files exist, the app launches the configure wizard automatically.

---

## Working with OpenAPI

Auth0 publishes the Management API spec at:
`https://auth0.com/docs/api/management/v2/openapi.yaml`

Use `oapi-codegen` to:

- Generate model structs from the spec (`oapi-codegen -generate types`)
- Generate mock servers for testing (`oapi-codegen -generate chi-server`)
- Keep the client in sync when spec changes

See `docs/OPENAPI_USAGE.md` for the full workflow.

---

## Key Principles

1. **Clean separation — no compromises:** `client/` owns Auth0 API communication. `modules/` owns domain logic. `tui/` owns presentation. **`modules/` does not import `tui/`** — ever. This is not a preference, it's a rule. If you find yourself needing to import the TUI from a module, the architecture is wrong — fix the design instead.
2. **TDD always:** No production code without a failing test first.
3. **Config is king:** Tenant context lives in config files. The app should never hardcode a domain or tenant.
4. **Extensible by design:** New modules follow the same pattern as existing ones — independent, testable, self-contained.
5. **Meaningful logs:** All operations write structured logs. The TUI has a log viewer with scrollable, expandable events exported as JSON.