# A0Hero

Auth0 tenant management, right in your terminal.

A0Hero wraps the Auth0 Management API in a fullscreen TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Browse users, applications, roles, connections, and logs — no web dashboard required.

## Quick Start

```bash
# Install (from release)
curl -sL https://github.com/samrocksc/a0hero/releases/latest/download/a0hero_<version>_darwin-arm64.tar.gz | tar xz
sudo mv a0hero /usr/local/bin/

# Or build from source
git clone https://github.com/samrocksc/a0hero.git
cd a0hero && make install && make build
```

### First Run

```bash
a0hero
```

If no tenant is configured, A0Hero launches the **Configure** tab automatically. Enter your Auth0 credentials:

| Field         | Where to find it                                             |
| ------------- | ------------------------------------------------------------ |
| Tenant Name   | Any label you want (e.g. `dev`, `prod`)                     |
| Domain        | Auth0 Dashboard → Settings → Domain                         |
| Client ID     | Auth0 Dashboard → Applications → your M2M app → Client ID   |
| Client Secret | Auth0 Dashboard → Applications → your M2M app → Client Secret |

The application needs a **Machine-to-Machine** app with Management API scopes (read:users, read:clients, read:roles, etc.).

Config is saved to `~/.config/a0hero/<name>.yaml` and reused on next launch.

### Environment Variables

You can also authenticate via env vars (takes precedence over config files):

```bash
export AUTH0_DOMAIN=your-tenant.auth0app.com
export AUTH0_CLIENT_ID=your-client-id
export AUTH0_CLIENT_SECRET=your-client-secret
a0hero
```

## Navigation

A0Hero is a **fullscreen tabbed app** — no extra keypresses to navigate:

| Key                   | Action                        |
| --------------------- | ----------------------------- |
| `tab` / `→` / `l`    | Next section                  |
| `shift+tab` / `←` / `h` | Previous section           |
| `↑` / `k`             | Scroll up                     |
| `↓` / `j`             | Scroll down                   |
| `enter`               | View item detail              |
| `esc`                 | Close detail / go back        |
| `q`                   | Quit                          |

Sections: **Users → Clients → Roles → Connections → Logs → Configure**

Content loads automatically when you switch tabs. No "select and confirm" needed.

## Debug Logging

```bash
a0hero --debug
```

Writes structured JSON logs to `logs/<date>.log` in the current directory. Useful for troubleshooting Auth0 API errors, token issues, etc.

## Version

```bash
a0hero --version
# a0hero v0.0.1 (commit: abc1234, built: 2026-04-14T12:00:00Z, darwin/arm64)
```

## Building from Source

```bash
make build                          # Current platform, version=dev
make build VERSION=v0.0.1           # Current platform, tagged version

# Cross-compile everything
make dist-all VERSION=v0.0.1

# Package release archives
make release-archives VERSION=v0.0.1
```

### Build Requirements

- Go 1.24+
- Make

## Architecture

```
cmd/a0hero/        CLI entry point (cobra)
client/            Auth0 API client (auth, config, HTTP transport)
models/            Shared types and interfaces
modules/           Domain modules (users, clients, roles, connections, logs)
tui/               Bubble Tea TUI layer
tui/components/    Shared UI components (table)
logger/            Structured debug logger
version/           Build version info (injected via ldflags)
tests/             Integration tests with mock HTTP server
```

**Key rule:** `modules/` never imports `tui/`. The domain layer and presentation layer are strictly separate.

## Releases

Releases are built by GitHub Actions on every `v*` tag push. Available platforms:

| Platform     | Archive                                |
| ------------ | -------------------------------------- |
| macOS ARM    | `a0hero_<version>_darwin-arm64.tar.gz` |
| macOS Intel  | `a0hero_<version>_darwin-amd64.tar.gz` |
| Linux ARM    | `a0hero_<version>_linux-arm64.tar.gz`  |
| Linux Intel  | `a0hero_<version>_linux-amd64.tar.gz`  |
| Windows Intel | `a0hero_<version>_windows-amd64.zip`  |

## License

MIT