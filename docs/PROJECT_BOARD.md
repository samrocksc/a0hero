# A0Hero — Project Board

**Last updated:** 2026-04-13

---

## Current Sprint

**Goal:** Establish the core infrastructure — `client/` package, config loading, auth, and the TUI shell — before building any modules.

---

## To Do

| # | Item | Module | Notes | Status |
|---|------|--------|-------|--------|
| 1 | Implement `client/` — config loading, auth, Auth0 API client | infra | Load `config/*.yaml`, client credentials auth, token caching, HTTP methods | ⬜ |
| 2 | Implement TUI shell — main menu, tenant switching, module entry points | tui | Bubble Tea app, main menu with module navigation, tenant selector, first-run wizard | ⬜ |
| 3 | `a0hero configure` command — Huh form, config write, connection test | cmd | Prompt for name, client_id, client_secret; write to `config/<name>.yaml` | ⬜ |
| 4 | Logs module — TDD, then implementation | logs | List with date filter, click-to-expand, JSON export | ⬜ |
| 5 | Clients module — TDD, then implementation | clients | List, get, create, update, update redirect_uris, delete | ⬜ |
| 6 | Users module — TDD, then implementation | users | List, get, search, create, update, delete | ⬜ |
| 7 | Roles module — TDD, then implementation | roles | List, get, create, update, delete, assign users, permissions | ⬜ |
| 8 | Connections module — TDD, then implementation | connections | List, get, create, update, delete | ⬜ |
| 9 | CI/CD setup — GitHub Actions | infra | Lint, test, build on PR/merge | ⬜ |

---

## In Progress

| # | Item | Who | Status | Notes |
|---|------|-----|--------|-------|

---

## Done ✓

| # | Item | Completed | Notes |
|---|------|-----------|-------|

---

## Notes

- Task numbers (#) are stable identifiers — don't renumber, just add new rows
- Use `⬜` for not started, `🔄` for in progress, `✅` for done
- Add new items to the bottom of **To Do**
- Move items to **In Progress** when assigned, to **Done** when verified by tester
- Move stalled or blocked items back to **To Do** with a note explaining why

---

## How to Use This Board

This is the single source of truth for the project's current state. Update it every time work starts, stops, or finishes. If we need to stop and come back later, this board tells us exactly where we are.

PM owns this board. When a work item is delivered and signed off, PM updates it to Done and moves to the next in To Do.

---

## Sprint Notes

**(2026-04-13)** — Project kicked off. Full spec defined with Sam. Docs written: AGENTS.md, AGENT_PERSONAS.md, ARCHITECTURE.md, MODULES.md, OPENAPI_USAGE.md, PROJECT.md. First work item is the `client/` package (config loading + auth). PM to assign to dev after architecture review.