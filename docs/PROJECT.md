# A0Hero — Project Status

## Current Status

**Stage:** Early — repo initialized, structure defined, docs being drafted.

**Last updated:** 2026-04-13

---

## What's Done

- [x] Repo initialized at `~/GitHub/work/a0hero/`
- [x] Go module initialized with `.gitignore`
- [x] `docs/` folder created with `ARCHITECTURE.md`, `MODULES.md`, `OPENAPI_USAGE.md`
- [x] `AGENTS.md` and `AGENT_PERSONAS.md` written with onboarding instructions and three personas

---

## What's Next

- [ ] Implement `client/` — config loading, auth, Auth0 API client
- [ ] Implement first module (logs) — TDD against mock, then real API
- [ ] Implement TUI shell — main menu, tenant switching, module navigation
- [ ] Implement `a0hero configure` — Huh form, config file write, connection test

---

## Decisions Made

| Decision | Resolution |
|----------|-----------|
| **Charm stack** | Bubble Tea + Huh + Lip Gloss for all TUI work |
| **Auth approach** | Client credentials stored in config.yaml. No device flow. |
| **Config structure** | One `.yaml` file per tenant in `config/` directory |
| **Module pattern** | `modules/<name>/client.go`, `model.go`, `view.go` — self-contained, independently testable |
| **TDD approach** | Tests driven by mock HTTP handlers generated from Auth0 OpenAPI spec via `oapi-codegen` |
| **Client separation** | `client/` is importable without the TUI. `modules/` does not import `tui/` |
| **Log export format** | JSON lines to file named `logs-<from>-to-<to>.json` |
| **First-run experience** | Auto-launch configure wizard if no `config/` files exist |
| **Tenant switching** | Reset all state — no carry-over from previous tenant context |

---

## Open Questions

| Question | Owner | Status |
|----------|-------|--------|
| Which Go TUI framework for CLI commands? (charm/cli vs cobra) | Sam | Open |
| Actions module — scope and priority? | Sam | Open |
| How to handle Auth0 rate limiting in logs viewer? | Sam | Open |
| CI/CD setup — GitHub Actions? | Sam | Open |
| Initial client credentials — what permissions does the app need? | Sam | Open |

---

## Agent Workflow Summary

```
Sam → PM → Architect (architecture feedback)
                 ↓
              PM → Sam (present plan)
                 ↓
Sam approves
                 ↓
PM → Tester (write failing TDD tests)
                 ↓
PM → Dev (implement)
                 ↓
Dev ↔ Tester (iterate until tests pass)
                 ↓
Tester → PM (sign-off)
                 ↓
PM → Sam (final deliverable)
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.24+ |
| TUI | Bubble Tea + Huh + Lip Gloss |
| Auth0 SDK | auth0-go or direct HTTP |
| CLI | charm/cli or cobra |
| Config | YAML via `gopkg.in/yaml.v3` |
| OpenAPI | oapi-codegen v2 |
| Testing | testify + mock HTTP handlers |

---

## Config File Format

```yaml
name: dev
domain: dev-tenant.auth0.com
client_id: YOUR_CLIENT_ID
client_secret: YOUR_CLIENT_SECRET
```

Auth can also be provided via `AUTH0_CLIENT_ID` and `AUTH0_CLIENT_SECRET` environment variables (takes precedence over config file).

---

## Key Files

| File | Purpose |
|------|---------|
| `AGENTS.md` | Agent onboarding guide — what this project is, how to work on it |
| `AGENT_PERSONAS.md` | Persona-specific system prompts (architect, PM, tester) |
| `docs/ARCHITECTURE.md` | Full architectural description — layers, patterns, directory structure |
| `docs/MODULES.md` | Per-module details — operations, models, view patterns |
| `docs/OPENAPI_USAGE.md` | How to use the Auth0 OpenAPI spec with oapi-codegen |
| `docs/PROJECT.md` | This file — status, decisions, open questions |