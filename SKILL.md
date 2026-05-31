`epd` is a Go CLI that drives Waveshare e-paper displays. It can operate directly on local hardware (Raspberry Pi GPIO) or push content to a remote display over gRPC.

> **Note:** The e-paper display is 800×480 pixels — there is very limited screen real estate. Keep body text short and concise. Avoid walls of text; prefer brief summaries, small tables, or short bullet lists.

## Prerequisites

- **Local mode**: A Raspberry Pi with a supported Waveshare e-paper display connected via GPIO.
- **Remote mode**: A reachable EPD daemon (`epd serve`) running on the display host.
- The `epd` binary installed on the machine issuing commands.

## Installation

### Pre-compiled binaries

Download from GitHub Releases: <https://github.com/justmiles/epd/releases>

- `epd_Linux_x86_64.tar.gz` — for standard Linux (amd64)
- `epd_Linux_armv6.tar.gz` — for Raspberry Pi (ARMv6)
- `epd_<version>_linux_armv6.deb` — Debian package for Raspberry Pi (installs systemd service)

### Debian package (Raspberry Pi)

```bash
sudo dpkg -i epd_*.deb
# Configures systemd service at /etc/systemd/system/epd.service
# Environment file at /etc/default/epd
```

## Architecture

```
┌──────────────────┐         gRPC (TCP :50051)         ┌──────────────────────┐
│  Client machine  │ ──────────────────────────────────▶│  Raspberry Pi        │
│  (any platform)  │                                    │  epd serve           │
│                  │                                    │  ├─ GPIO → e-Paper   │
└──────────────────┘                                    └──────────────────────┘
```

When `--device` is a `host:port` string the CLI auto-detects remote mode and sends commands over gRPC. Otherwise it talks to local hardware.

## Global Flags

| Flag | Short | Env Var | Default | Description |
|------|-------|---------|---------|-------------|
| `--device` | `-d` | `EPD_DEVICE` | `epd7in5v2` | Device type (local) or `host:port` (remote) |
| `--initialize` | `-i` | — | `false` | Wake the device from sleep before updating |
| `--sleep` | `-s` | — | `false` | Put the device to sleep after updating |
| `--debug` | — | — | `false` | Enable debug logging |

## Commands

### `epd display-image <path>`

Display a PNG/image file on the EPD.

```bash
# Local
epd display-image dashboard.png

# Remote
epd display-image --device pi.local:50051 dashboard.png
```

| Flag | Default | Description |
|------|---------|-------------|
| `--preview` | `false` | Generate/process but don't push to display |

### `epd display-text <text...>`

Render and display plain text.

```bash
epd display-text "Hello World"
epd display-text --device pi.local:50051 "Status: OK"
```

### `epd clear`

Clear the display to white.

```bash
epd clear
epd clear --device pi.local:50051
```

### `epd refresh-dashboard`

Generate and display a dashboard image with header, body, and optional weather.

The body supports **GitHub Flavored Markdown** (tables, task lists, bold, italic, strikethrough, code, etc.) and can be either inline text or a path to a `.md` file.

```bash
# Inline text
epd refresh-dashboard --header-text "My Dashboard" --body-text "Hello, world!"

# Markdown file
epd refresh-dashboard --header-text "TODOs" --body-text ./TODOs.md

# Remote display
epd refresh-dashboard --header-text "TODOs" --body-text TODOs.md --device pi.local:50051

# Preview only (writes dashboard-image.png without pushing to display)
epd refresh-dashboard --header-text "Test" --body-text "Preview" --preview
```

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--header-text` | `EPD_HEADER_TEXT` | `""` | Dashboard header text |
| `--body-text` | `EPD_BODY_TEXT` | `""` | Body text or path to a `.md` file |
| `--weather-api-key` | `EPD_WEATHER_API_KEY` | `""` | OpenWeatherMap API key |
| `--weather-zip` | `EPD_WEATHER_ZIP` | `60601` | ZIP code for weather |
| `--weather-country` | `EPD_WEATHER_COUNTRY` | `US` | Country code |
| `--weather-language` | `EPD_WEATHER_LANGUAGE` | `EN` | Weather language |
| `--weather-temp-unit` | `EPD_WEATHER_TEMP_UNIT` | `F` | Temperature unit (`F` or `C`) |
| `--location` | `EPD_LOCATION` | `America/Chicago` | IANA timezone for date display |
| `--preview` | — | `false` | Write `dashboard-image.png` locally without pushing to display |
| `--tududi-url` | `TUDUDI_URL` | `""` | Tududi base URL (e.g. `http://localhost:3004`) |
| `--tududi-api-token` | `TUDUDI_API_TOKEN` | `""` | Tududi API bearer token |
| `--tududi-project` | `TUDUDI_PROJECT` | `""` | Show only tasks in this project (ID or name) |
| `--tududi-max-tasks` | `TUDUDI_MAX_TASKS` | `15` | Max tasks to display |

> **Tududi priority:** `--body-text` takes precedence over Tududi. Set only `TUDUDI_URL` + `TUDUDI_API_TOKEN` (and leave `--body-text` unset) to use Tududi as the body source.

## Environment Variables

All environment variables and their corresponding flags:

| Variable | Flag | Description |
|----------|------|-------------|
| `EPD_DEVICE` | `--device` | Device type or remote `host:port` |
| `EPD_HEADER_TEXT` | `--header-text` | Dashboard header |
| `EPD_BODY_TEXT` | `--body-text` | Dashboard body (text or file path) |
| `EPD_WEATHER_API_KEY` | `--weather-api-key` | OpenWeatherMap key |
| `EPD_WEATHER_ZIP` | `--weather-zip` | Weather ZIP code |
| `EPD_WEATHER_COUNTRY` | `--weather-country` | Country code |
| `EPD_WEATHER_LANGUAGE` | `--weather-language` | Language |
| `EPD_WEATHER_TEMP_UNIT` | `--weather-temp-unit` | Temp unit |
| `EPD_LOCATION` | `--location` | IANA timezone |
| `TUDUDI_URL` | `--tududi-url` | Tududi base URL |
| `TUDUDI_API_TOKEN` | `--tududi-api-token` | Bearer token from Tududi Settings → API Tokens |
| `TUDUDI_PROJECT` | `--tududi-project` | Project ID (integer) or name substring |
| `TUDUDI_MAX_TASKS` | `--tududi-max-tasks` | Max tasks to show (default 15) |

## Typical Agent Workflows

### 1. Push content to a remote EPD

```bash
# Display a generated image
epd display-image --device <host>:50051 output.png

# Display a text message
epd display-text --device <host>:50051 "Build passed ✓"

# Refresh the dashboard
epd refresh-dashboard --device <host>:50051 --header-text "System Status" --body-text "path/to/markdown/file.md"
```

### 2. Generate and display a Markdown dashboard

```bash
# Write markdown content to a temp file, then display it
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

### 3. Preview a dashboard without a display

```bash
epd refresh-dashboard \
  --header-text "Preview" \
  --body-text content.md \
  --preview
# Outputs: dashboard-image.png in the current directory
```

### 4. Clear the display

```bash
epd clear --device <host>:50051
```

### 5. Show Tududi tasks on the display

```bash
# Set credentials (or use env vars)
export TUDUDI_URL="http://localhost:3004"
export TUDUDI_API_TOKEN="your-api-token"

# All active tasks
epd refresh-dashboard --header-text "Tasks" --device <host>:50051

# Tasks filtered to a project
epd refresh-dashboard --header-text "Work" --tududi-project "Work" --device <host>:50051

# Preview locally
epd refresh-dashboard --header-text "Tasks" --preview
```

## Supported Displays

| Model | Tested Hardware |
|-------|-----------------|
| `epd7in5v2` | Raspberry Pi Zero |
