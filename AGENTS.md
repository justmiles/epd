# AGENTS.md — `epd` Project Guide

> This file is the authoritative reference for AI agents working in this repository.
> Read it in full before making any changes.

---

## What This Project Is

`epd` is a Go CLI that drives **Waveshare e-paper displays** (EPDs). It can:
- Talk directly to a locally connected display via Raspberry Pi GPIO.
- Push content to a remote display over **gRPC** (TCP `:50051`).

**Display resolution: 800 × 480 pixels.** This is very limited screen space. Always keep content short and concise. No walls of text — prefer brief summaries, small tables, or compact bullet lists.

---

## Repository Layout

```
epd/
├── main.go                   # Entry point — calls cmd.Execute()
├── cmd/                      # Cobra CLI commands
│   ├── root.go               # Global flags: --device, --initialize, --sleep, --debug
│   ├── clear.go              # epd clear
│   ├── image.go              # epd display-image
│   ├── text.go               # epd display-text
│   ├── refresh-dashboard.go  # epd refresh-dashboard
│   └── serve.go              # epd serve (gRPC daemon)
├── lib/
│   ├── dashboard/            # Dashboard image generation (header + GFM body + weather)
│   │   ├── dashboard.go      # Core rendering logic
│   │   ├── epd.go            # EPDOptions / WithEPD option
│   │   ├── openweather.go    # OpenWeatherMap API integration
│   │   ├── icons.go          # Weather icon rendering
│   │   └── utilities.go      # Image/font helpers
│   ├── display/              # Display abstraction layer
│   │   ├── display.go        # Service interface (DisplayImage, DisplayText, Clear, Sleep, Close)
│   │   ├── local.go          # LocalDisplay — GPIO via go-rpio
│   │   └── remote.go         # RemoteDisplay — gRPC client
│   ├── epd/                  # Low-level EPD device drivers
│   ├── epd7in5v2/            # Driver for the 7.5" v2 Waveshare display
│   ├── md-png/               # Markdown → PNG renderer (uses goldmark + gg)
│   ├── tududi/               # Tududi task provider (HTTP client + Markdown formatter)
│   └── server/               # gRPC server implementation
├── proto/
│   ├── epd.proto             # Protobuf service definition
│   └── epdpb/               # Generated Go protobuf/gRPC code (do not edit)
├── packaging/                # Systemd service file + Debian packaging scripts
├── devbox.json               # Devbox dev environment
├── .goreleaser.yaml          # Release pipeline (linux/amd64 + linux/armv6 + .deb)
├── .drone.yml                # CI: goreleaser on git tag push
└── Makefile                  # make build / make proto
```

---

## Architecture

```
┌──────────────────┐       gRPC (TCP :50051)       ┌──────────────────────┐
│  Client machine  │ ─────────────────────────────▶ │  Raspberry Pi        │
│  (any platform)  │                                 │  epd serve           │
│                  │                                 │  ├─ GPIO → e-Paper   │
└──────────────────┘                                 └──────────────────────┘
```

- When `--device` is a `host:port` string, the CLI auto-detects **remote mode** and communicates over gRPC.
- Otherwise it drives **local hardware** via GPIO.
- The `display.Service` interface abstracts both backends — new commands should use it rather than calling local or remote code directly.

---

## Key Abstractions

### `display.Service` interface (`lib/display/display.go`)
```go
type Service interface {
    DisplayImage(pngData []byte) error
    DisplayText(text string) error
    Clear() error
    Sleep() error
    Close() error
}
```
Use `display.IsRemote(device)` to branch between local and remote paths. `LocalDisplay` and `RemoteDisplay` both satisfy this interface.

### `dashboard.Dashboard` (`lib/dashboard/dashboard.go`)
Creates a 800×480 PNG from:
- A **header** text line (date, header text, optional weather widget)
- A **body** rendered from GitHub Flavored Markdown (goldmark)

Constructed with functional options:
```go
d, err := dashboard.NewDashboard(
    dashboard.WithWeatherAPI(&weatherAPIOptions),
)
err = d.Generate("output.png", headerText, bodyText)
```

### `newDisplayService` (`cmd/root.go` area)
Factory used by commands to obtain the correct `display.Service` (local vs. remote). Always use this; don't construct `LocalDisplay` or `RemoteDisplay` directly in command code.

---

## CLI Reference

### Global Flags

| Flag | Short | Env Var | Default | Description |
|------|-------|---------|---------|-------------|
| `--device` | `-d` | `EPD_DEVICE` | `epd7in5v2` | Device type or `host:port` for remote |
| `--initialize` | `-i` | — | `false` | Wake device from sleep before update |
| `--sleep` | `-s` | — | `false` | Put device to sleep after update |
| `--debug` | — | — | `false` | Enable debug logging |

### Commands

| Command | Description |
|---------|-------------|
| `epd clear` | Clear display to white |
| `epd display-image <path>` | Display a PNG file |
| `epd display-text <text...>` | Render and display plain text |
| `epd refresh-dashboard` | Generate and push a dashboard (header + GFM body + weather) |
| `epd serve` | Run gRPC daemon on the Raspberry Pi (`--port`, default `50051`) |

#### `refresh-dashboard` flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--header-text` | `EPD_HEADER_TEXT` | `""` | Header line text |
| `--body-text` | `EPD_BODY_TEXT` | `""` | Inline text **or** path to a `.md` file |
| `--weather-api-key` | `EPD_WEATHER_API_KEY` | `""` | OpenWeatherMap API key |
| `--weather-zip` | `EPD_WEATHER_ZIP` | `60601` | ZIP code |
| `--weather-country` | `EPD_WEATHER_COUNTRY` | `US` | Country code |
| `--weather-language` | `EPD_WEATHER_LANGUAGE` | `EN` | Weather language |
| `--weather-temp-unit` | `EPD_WEATHER_TEMP_UNIT` | `F` | `F` or `C` |
| `--location` | `EPD_LOCATION` | `America/Chicago` | IANA timezone for date display |
| `--preview` | — | `false` | Write `dashboard-image.png` locally; do not push to display |
| `--tududi-url` | `TUDUDI_URL` | `""` | Tududi base URL (e.g. `http://localhost:3004`) |
| `--tududi-api-token` | `TUDUDI_API_TOKEN` | `""` | Tududi API bearer token |
| `--tududi-project` | `TUDUDI_PROJECT` | `""` | Show only tasks in this project (ID or name) |
| `--tududi-max-tasks` | `TUDUDI_MAX_TASKS` | `15` | Max tasks to display |

---

## Environment Variables

| Variable | Flag equivalent | Notes |
|----------|-----------------|-------|
| `EPD_DEVICE` | `--device` | Set to `host:port` for remote mode |
| `EPD_HEADER_TEXT` | `--header-text` | |
| `EPD_BODY_TEXT` | `--body-text` | Inline text or file path |
| `EPD_WEATHER_API_KEY` | `--weather-api-key` | From openweathermap.org |
| `EPD_WEATHER_ZIP` | `--weather-zip` | |
| `EPD_WEATHER_COUNTRY` | `--weather-country` | |
| `EPD_WEATHER_LANGUAGE` | `--weather-language` | |
| `EPD_WEATHER_TEMP_UNIT` | `--weather-temp-unit` | |
| `EPD_LOCATION` | `--location` | IANA timezone string |
| `TUDUDI_URL` | `--tududi-url` | Tududi base URL |
| `TUDUDI_API_TOKEN` | `--tududi-api-token` | Bearer token from Tududi Settings → API Tokens |
| `TUDUDI_PROJECT` | `--tududi-project` | Project ID (integer) or name substring |
| `TUDUDI_MAX_TASKS` | `--tududi-max-tasks` | Defaults to 15 |

> **Dev environment**: `.envrc` (loaded by direnv/devbox) pre-sets `EPD_DEVICE` to the Tailscale IP of the Pi and a weather API key. Do not commit secrets.

---

## Agent Workflows

### Push content to the remote display

```bash
# Display a generated PNG
epd display-image --device <host>:50051 output.png

# Display a short text message
epd display-text --device <host>:50051 "Build passed ✓"

# Refresh dashboard from a Markdown file
epd refresh-dashboard --device <host>:50051 \
  --header-text "System Status" \
  --body-text /tmp/status.md
```

### Generate a Markdown dashboard and push remotely

```bash
cat > /tmp/status.md << 'EOF'
| Service | Status |
|---------|--------|
| API     | ✅ Up  |
| DB      | ✅ Up  |
| Cache   | ⚠️ Degraded |
EOF

epd refresh-dashboard \
  --device <host>:50051 \
  --header-text "System Status" \
  --body-text /tmp/status.md
```

### Preview without a physical display

```bash
epd refresh-dashboard \
  --header-text "Preview" \
  --body-text content.md \
  --preview
# Writes: dashboard-image.png in the current directory
```

### Clear the display

```bash
epd clear --device <host>:50051
```

### Push Tududi tasks to the display

```bash
# Show all active tasks from Tududi
epd refresh-dashboard \
  --device <host>:50051 \
  --header-text "Tasks" \
  --tududi-url "$TUDUDI_URL" \
  --tududi-api-token "$TUDUDI_API_TOKEN"

# Filter to a specific project by name
epd refresh-dashboard \
  --device <host>:50051 \
  --header-text "Work Tasks" \
  --tududi-url "$TUDUDI_URL" \
  --tududi-api-token "$TUDUDI_API_TOKEN" \
  --tududi-project "Work"

# --body-text takes precedence over Tududi when both are set
epd refresh-dashboard \
  --header-text "Override" \
  --body-text "Custom content" \
  --tududi-url "$TUDUDI_URL" \
  --tududi-api-token "$TUDUDI_API_TOKEN" \
  --preview
# ↑ Shows "Custom content", not Tududi tasks
```

---

## Build & Development

```bash
# Build the binary
make build          # go build -o epd .

# Regenerate protobuf/gRPC stubs (requires protoc)
make proto

# Release (CI only — triggered on git tag v*)
# Uses goreleaser: produces linux/amd64 tar.gz, linux/armv6 tar.gz, and .deb
```

### Dev Environment (devbox)

```bash
devbox shell        # Activates the pinned Go toolchain
```

`devbox.json` pins all tooling versions. Use devbox rather than system Go to ensure reproducibility.

---

## Adding a New Command

1. Create `cmd/<name>.go` in the `cmd` package.
2. Register it in `init()` via `rootCmd.AddCommand(myCmd)`.
3. Use `newDisplayService(device, initialize)` to get a `display.Service`.
4. Use `resolveTextOrFile(s)` if your command accepts text-or-file inputs.
5. Respect the global `--sleep` flag: call `svc.Sleep()` when `sleep == true`.
6. Keep displayed content within **800 × 480 px**. Test with `--preview`.

---

## Adding a New Display Driver

1. Implement the hardware driver in `lib/epd<model>/`.
2. Register it in `lib/display/local.go` (the device-type switch).
3. Add the model to the **Supported Displays** table in `README.md` and `SKILL.md`.
4. Update `TODO.md` device compatibility table.

---

## Content Guidelines

- **Screen size**: 800 × 480 px — very limited real estate.
- **Body text**: Prefer compact Markdown. GitHub Flavored Markdown is fully supported (tables, task lists, bold, italic, strikethrough, inline code).
- **Header text**: Single short line — used as the dashboard title alongside the date.
- **No images in body**: The body renderer converts Markdown to a rasterized PNG; embed icons via the weather widget, not arbitrary images.
- **Weather widget**: Requires a valid `EPD_WEATHER_API_KEY`. Omit the flag to disable it.

---

## Supported Displays

| Model | Status | Notes |
|-------|:------:|-------|
| `epd7in5v2` | ✅ Tested | Primary dev device (Raspberry Pi Zero) |
| `epd4in2` | Planned | Awaiting hardware |
| `epd2in13` | Planned | Compact layout needed |

---

## Release Process

Releases are automated via **Drone CI** (`.drone.yml`) on `v*` tag pushes:
- `goreleaser release --clean` builds all targets defined in `.goreleaser.yaml`.
- Artifacts: `epd_Linux_x86_64.tar.gz`, `epd_Linux_armv6.tar.gz`, `epd_<version>_linux_armv6.deb`.
- The `.deb` installs a systemd service (`/etc/systemd/system/epd.service`) and an environment file (`/etc/default/epd`).
