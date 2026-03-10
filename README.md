# CareLink Go

[![CI](https://github.com/mrowling/carelink-go/actions/workflows/ci.yml/badge.svg)](https://github.com/mrowling/carelink-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/mrowling/carelink-go?include_prereleases)](https://github.com/mrowling/carelink-go/releases/latest)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io-blue?logo=docker)](https://github.com/mrowling/carelink-go/pkgs/container/carelink-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrowling/carelink-go)](https://goreportcard.com/report/github.com/mrowling/carelink-go)

A standalone Go implementation of the CareLink bridge that fetches Medtronic pump and CGM data from the CareLink API and serves it via HTTP endpoints.

## Features

- **Standalone HTTP server** - Single executable with built-in web API
- **Background polling** - Continuously fetches data from CareLink API
- **HTTP API** - Query latest glucose readings via REST endpoints
- **Health checks** - Kubernetes-ready liveness/readiness probes
- **Automatic token refresh** - OAuth2 tokens refreshed automatically
- **SQLite database** - Local storage of all glucose readings and device status
- **Structured logging** - Configurable log levels (DEBUG/INFO/WARN/ERROR)
- Full feature parity with Node.js version:
  - Patient and care partner account support
  - BLE device detection and handling
  - Multiple API endpoint fallback logic
  - Proxy rotation support
  - Retry logic with exponential backoff
  - Blood glucose conversion (mg/dL to mmol/L)

## Prerequisites

- Go 1.21 or later (for building from source)
- Valid CareLink account credentials
- Initial authentication setup (see below)

## Installation

### Using Docker (Recommended)

Pre-built multi-architecture Docker images are available from GitHub Container Registry:

```bash
# Pull the latest version
docker pull ghcr.io/mrowling/carelink-go:latest

# Or pull a specific version
docker pull ghcr.io/mrowling/carelink-go:1.0.0

# Run the container
docker run -d \
  --name carelink-go \
  -p 8080:8080 \
  -v ~/.carelink/config:/config:ro \
  -v ~/.carelink/data:/data \
  -e CARELINK_CONFIG_DIR=/config \
  -e CARELINK_DATA_DIR=/data \
  ghcr.io/mrowling/carelink-go:latest
```

**Supported architectures:**
- `linux/amd64` - Standard x86_64 Linux systems
- `linux/arm64` - ARM-based systems (Raspberry Pi, Apple Silicon via Docker Desktop)

### Pre-built Binaries

Download pre-compiled binaries from the [GitHub Releases](https://github.com/mrowling/carelink-go/releases) page.

**Linux:**
```bash
# Download and install (replace VERSION with the desired version, e.g., v1.0.0)
curl -LO https://github.com/mrowling/carelink-go/releases/download/VERSION/carelink-go-linux-amd64.tar.gz
tar xzf carelink-go-linux-amd64.tar.gz
chmod +x carelink-go-linux-amd64
sudo mv carelink-go-linux-amd64 /usr/local/bin/carelink-go
```

**macOS:**
```bash
# For Apple Silicon (M1/M2/M3)
curl -LO https://github.com/mrowling/carelink-go/releases/download/VERSION/carelink-go-darwin-arm64.tar.gz
tar xzf carelink-go-darwin-arm64.tar.gz
chmod +x carelink-go-darwin-arm64
sudo mv carelink-go-darwin-arm64 /usr/local/bin/carelink-go

# For Intel Macs
curl -LO https://github.com/mrowling/carelink-go/releases/download/VERSION/carelink-go-darwin-amd64.tar.gz
tar xzf carelink-go-darwin-amd64.tar.gz
chmod +x carelink-go-darwin-amd64
sudo mv carelink-go-darwin-amd64 /usr/local/bin/carelink-go
```

**Windows:**

Download the `.zip` file from the [releases page](https://github.com/mrowling/carelink-go/releases) and extract it.

### From Source

```bash
git clone https://github.com/mrowling/carelink-go.git
cd carelink-go
go build -o carelink-go
```

### Using go install

```bash
go install github.com/mrowling/carelink-go@latest
```

## Authentication Setup

CareLink Go requires initial authentication using the Node.js version of carelink-bridge.

**One-time setup:**

1. Clone and setup the carelini-bridge repository:
   ```bash
   git clone https://github.com/mrowling/carelink-bridge.git
   cd carelink-bridge
   npm install
   ```

2. Run the login command:
   ```bash
   npm run login
   ```

3. Copy the generated `logindata.json` to the config directory:
   - `~/.carelink/config/logindata.json` **(recommended)**
   - Same directory as the carelink-go binary

**After initial setup:**

CareLink Go will automatically refresh tokens, so you only need to do this once. Tokens typically remain valid for 30-90 days of continuous use.

## Configuration

### Directory Structure

CareLink Go uses separate directories for configuration and data:

**Config directory** (default: `~/.carelink/config/`):
- `.env` - Main configuration
- `my.env` - Optional overrides
- `logindata.json` - OAuth tokens
- `https.txt` - Proxy list (optional)

**Data directory** (default: `~/.carelink/data/`):
- `carelink.db` - SQLite database

This separation allows independent mounting in containerized environments (Docker, Kubernetes).

### Environment Variables

**Directory Configuration:**
```env
CARELINK_CONFIG_DIR=/path/to/config  # Default: ~/.carelink/config
CARELINK_DATA_DIR=/path/to/data      # Default: ~/.carelink/data
CARELINK_DB_PATH=/custom/db.db       # Overrides data_dir/carelink.db
```

**HTTP Server Configuration:**
```env
CARELINK_PORT=8080                      # HTTP server port (default: 8080)
CARELINK_LOG_LEVEL=INFO                 # DEBUG, INFO, WARN, ERROR (default: INFO)
CARELINK_HEALTH_CHECK_STALE_MINS=15     # Minutes before health check fails (default: 15)
```

**Application Configuration (.env file):**

```env
CARELINK_USERNAME=your_username
CARELINK_PASSWORD=your_password
MMCONNECT_SERVER=EU  # or US
MMCONNECT_COUNTRYCODE=gb
MMCONNECT_LANGCODE=en
CARELINK_PATIENT_ID=  # Optional, for care partner accounts
CARELINK_FETCH_INTERVAL=300  # seconds
CARELINK_VERBOSE=false
USE_PROXY=false
```

Place `.env` in one of these locations (checked in order):
1. Same directory as the binary
2. Config directory (`~/.carelink/config/` or `CARELINK_CONFIG_DIR`)

### Kubernetes / Docker Configuration

For containerized deployments, mount directories separately:

**Kubernetes Example with Health Checks:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: carelink-go
spec:
  containers:
  - name: carelink
    image: ghcr.io/mrowling/carelink-go:latest
    ports:
    - containerPort: 8080
      name: http
    env:
    - name: CARELINK_CONFIG_DIR
      value: /config
    - name: CARELINK_DATA_DIR
      value: /data
    - name: CARELINK_PORT
      value: "8080"
    - name: CARELINK_LOG_LEVEL
      value: INFO
    - name: CARELINK_HEALTH_CHECK_STALE_MINS
      value: "15"
    volumeMounts:
    - name: config
      mountPath: /config
      readOnly: true
    - name: data
      mountPath: /data
    livenessProbe:
      httpGet:
        path: /health
        port: http
      initialDelaySeconds: 30
      periodSeconds: 60
      timeoutSeconds: 5
      failureThreshold: 3
    readinessProbe:
      httpGet:
        path: /health
        port: http
      initialDelaySeconds: 10
      periodSeconds: 30
      timeoutSeconds: 5
      failureThreshold: 2
  volumes:
  - name: config
    configMap:
      name: carelink-config  # Contains .env, logindata.json
  - name: data
    persistentVolumeClaim:
      claimName: carelink-data  # Persistent storage for database
```

**Docker Example:**

```bash
docker run -d \
  -e CARELINK_CONFIG_DIR=/config \
  -e CARELINK_DATA_DIR=/data \
  -e CARELINK_PORT=8080 \
  -e CARELINK_LOG_LEVEL=INFO \
  -p 8080:8080 \
  -v /host/config:/config:ro \
  -v /host/data:/data \
  ghcr.io/mrowling/carelink-go:latest
```

### Database Location

Default: `~/.carelink/data/carelink.db`

Override with environment variable:
```bash
CARELINK_DB_PATH=/custom/path/carelink.db ./carelink-go poll
```

## Usage

### Starting the Server

The default behavior starts both the HTTP server and background polling:

```bash
./carelink-go
```

This will:
1. Start an HTTP server on port 8080 (configurable via `CARELINK_PORT`)
2. Begin polling the CareLink API every 300 seconds (configurable via `CARELINK_FETCH_INTERVAL`)
3. Store data in the SQLite database
4. Serve the latest data via HTTP endpoints

**Run on custom port:**
```bash
CARELINK_PORT=3000 ./carelink-go
```

**Enable debug logging:**
```bash
CARELINK_LOG_LEVEL=DEBUG ./carelink-go
```

### HTTP API Endpoints

#### GET /latest - Latest Glucose Reading

Returns the most recent glucose reading from the database.

**Example request:**
```bash
curl http://localhost:8080/latest
```

**Response (200 OK):**
```json
{
  "sgv_mmol": 12.3,
  "direction": "Flat",
  "trend": 4,
  "age_minutes": 5
}
```

**Fields:**
- `sgv_mmol` - Blood glucose in mmol/L (1 decimal place)
- `direction` - Trend direction string (omitted if not available)
- `trend` - Numeric trend value 0-9 (omitted if not available)
- `age_minutes` - Minutes since the reading

**Trend values:** 
- 0 = NONE
- 1 = TripleUp/DoubleUp
- 2 = SingleUp
- 3 = FortyFiveUp
- 4 = Flat
- 5 = FortyFiveDown
- 6 = SingleDown
- 7 = DoubleDown/TripleDown

**Error response (404 Not Found):**
```json
{
  "error": "No glucose data found"
}
```

#### GET /health - Health Check

Returns the health status of the service. Suitable for Kubernetes liveness and readiness probes.

**Example request:**
```bash
curl http://localhost:8080/health
```

**Response (200 OK) - Healthy:**
```json
{
  "status": "healthy",
  "last_fetch": "2026-03-10T17:09:45Z",
  "last_fetch_age_minutes": 5,
  "database_entries": 1234
}
```

**Response (503 Service Unavailable) - Unhealthy:**
```json
{
  "status": "unhealthy",
  "reason": "No data fetched yet",
  "database_entries": 0
}
```

or

```json
{
  "status": "unhealthy",
  "reason": "Data is stale (20 minutes old)",
  "last_fetch": "2026-03-10T16:50:00Z",
  "last_fetch_age_minutes": 20,
  "database_entries": 1234
}
```

The health check fails (returns 503) if:
- No data has been fetched yet
- The last successful fetch was more than 15 minutes ago (configurable via `CARELINK_HEALTH_CHECK_STALE_MINS`)

#### GET / - Root

Returns 204 No Content (empty response).

### Logging

Configure logging via the `CARELINK_LOG_LEVEL` environment variable:

- `DEBUG` - Verbose output including full JSON API responses
- `INFO` - Standard operational messages (default)
- `WARN` - Warning messages only
- `ERROR` - Error messages only

**Example DEBUG output:**
```
[INFO] [Server] Starting HTTP server on :8080
[INFO] [Poller] Starting with interval 300s
[INFO] [Poller] Fetching data from CareLink API
[DEBUG] [Poller] Raw API response: {"sgs":[...],"markers":[],...}
[INFO] [Poller] Data fetched successfully: 24 glucose entries, 1 device status
[INFO] [DB] Saved 24 glucose entries
[INFO] [DB] Saved 1 device status entries
```

### Help Command

```bash
./carelink-go help
```

## SQLite Database

Default location: `~/.carelink/data/carelink.db`

### Querying

```bash
# Recent readings
sqlite3 ~/.carelink/data/carelink.db "SELECT date_string, sgv, sgv_mmol, direction FROM glucose_entries ORDER BY date DESC LIMIT 10;"

# Export to CSV
sqlite3 -header -csv ~/.carelink/data/carelink.db "SELECT * FROM glucose_entries;" > glucose_export.csv
```

## Proxy Support

1. Set `USE_PROXY=true` in `.env`
2. Create `https.txt` in config directory with one proxy per line:
   ```
   ip:port
   ip:port:username:password
   ```
3. Place in `~/.carelink/config/https.txt` or current directory

## Troubleshooting

### "No logindata.json found"

Run the authentication setup using the Node.js version (see Authentication Setup above).

### "Token refresh failed"

Re-authenticate:
```bash
cd carelink-bridge  # Node.js version
npm run login
cp logindata.json ~/.carelink/config/
```

### "Database not found"

The database is created automatically on first run. Just start the server:
```bash
./carelink-go
# Database will be created and polling will begin
```

## Migration Guide

If you're upgrading from an older version that used `poll` and `latest` commands:

### What Changed

**Before (v1.x):**
```bash
# Start polling (outputs JSON to stdout)
./carelink-go poll

# Query latest reading (outputs JSON to stdout)
./carelink-go latest
```

**After (v2.0+):**
```bash
# Start server (polling runs in background)
./carelink-go

# Query latest reading via HTTP
curl http://localhost:8080/latest
```

### Benefits of New Architecture

1. **Single process** - No need to run separate poll/latest commands
2. **HTTP API** - Query data from any programming language or tool
3. **Health checks** - Built-in support for Kubernetes/Docker monitoring
4. **Structured logging** - Better visibility with configurable log levels
5. **Concurrent access** - Multiple clients can query /latest simultaneously

### Docker/Kubernetes Changes

**Old command:**
```yaml
command: ["./carelink-go", "poll"]
```

**New command:**
```yaml
# Just run the binary (no arguments)
command: ["./carelink-go"]

# Add health checks
livenessProbe:
  httpGet:
    path: /health
    port: 8080
```

## Development

### Task Runner

This project uses [Task](https://taskfile.dev) for common development tasks. Task is a modern alternative to Make with better usability.

**Install Task:**
```bash
# macOS
brew install go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin

# Windows (with Scoop)
scoop install task

# Or via Go
go install github.com/go-task/task/v3/cmd/task@latest
```

**Common tasks:**

```bash
# List all available tasks
task

# Build the application
task build

# Build with optimizations (smaller binary)
task build:optimized

# Build for all platforms (Linux, macOS, Windows)
task build:all

# Run the application
task run

# Run in development mode with debug logging
task dev

# Run all checks (format, vet, lint, test)
task check

# Format code
task fmt

# Run tests
task test

# Run tests with coverage
task test:coverage

# Setup development environment
task setup

# Clean build artifacts
task clean

# View recent glucose readings from database
task db:recent

# Export database to CSV
task db:export

# Check health endpoint (requires app to be running)
task health

# Get latest glucose reading (requires app to be running)
task latest
```

For a full list of available tasks, run `task --list`.

### Building

```bash
# Standard build
go build -o carelink-go

# Optimized build
go build -ldflags="-s -w" -o carelink-go

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o carelink-go-linux-amd64
```

### Dependencies

- `github.com/joho/godotenv` - .env file parsing
- `modernc.org/sqlite` - Pure Go SQLite driver

## License

MIT License - Same as the original carelink-bridge project.

Copyright (c) 2025 Domien
Copyright (c) 2026 Mitchell Rowling

## Credits

Based on [carelink-bridge](https://github.com/domien-f/carelink-bridge) by Domien.

Go implementation by Mitchell Rowling ([mrowling](https://github.com/mrowling)).
