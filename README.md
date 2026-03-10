# CareLink Go

A standalone Go implementation of the CareLink bridge that fetches Medtronic pump and CGM data from the CareLink API.

## Features

- **Standalone binary** - Single executable, no runtime dependencies
- **Automatic token refresh** - OAuth2 tokens refreshed automatically
- **SQLite database** - Local storage of all glucose readings and device status
- **Two operation modes**:
  - `poll` - Continuously fetch data from CareLink API
  - `latest` - Query most recent glucose reading from database
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

1. Clone and setup the original repository:
   ```bash
   git clone https://github.com/domien-f/carelink-bridge.git
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
**Kubernetes Example:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: carelink-go
spec:
  containers:
  - name: carelink
    image: carelink-go:latest
    env:
    - name: CARELINK_CONFIG_DIR
      value: /config
    - name: CARELINK_DATA_DIR
      value: /data
    volumeMounts:
    - name: config
      mountPath: /config
      readOnly: true
    - name: data
      mountPath: /data
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
  -v /host/config:/config:ro \
  -v /host/data:/data \
  carelink-go:latest poll
```

### Database Location

Default: `~/.carelink/data/carelink.db`

Override with environment variable:
```bash
CARELINK_DB_PATH=/custom/path/carelink.db ./carelink-go poll
```

## Usage

### Commands

#### poll - Fetch Data Continuously

```bash
./carelink-go poll
```

Continuously fetches data from CareLink API at the configured interval (default: 300 seconds).

**Output format:**

```json
{
  "devicestatus": [...],
  "entries": [
    {
      "type": "sgv",
      "sgv": 230,
      "sgv_mmol": 12.8,
      "date": 1773111180000,
      "dateString": "2026-03-10T02:53:00Z",
      "device": "connect-ngp",
      "direction": "NONE"
    }
  ]
}
```

#### latest - Query Latest Reading

```bash
./carelink-go latest
```

**Output format:**

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

**Trend values:** 0=NONE, 1=TripleUp/DoubleUp, 2=SingleUp, 3=FortyFiveUp, 4=Flat, 5=FortyFiveDown, 6=SingleDown, 7=DoubleDown/TripleDown

**Examples:**

```bash
# Get latest reading
./carelink-go latest

# Extract glucose value with jq
./carelink-go latest | jq '.sgv_mmol'

# Check if reading is recent
./carelink-go latest | jq 'select(.age_minutes < 10)'
```

#### help - Show Help

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

Run `poll` command once to create the database:
```bash
./carelink-go poll
# Wait for one fetch, then Ctrl+C
```

## Development

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

## Credits

Based on [carelink-bridge](https://github.com/domien-f/carelink-bridge) by Domien.

Go implementation by [mrowling](https://github.com/mrowling).
