# SecureLens Device Agent

A lightweight Go binary that scans local file systems for PII and sensitive data, reporting findings to SecureLens.

## Features

- **20+ PII Types**: Detects emails, SSNs, credit cards, API keys, AWS credentials, JWTs, private keys, database URLs, and more
- **Fast Scanning**: 4M+ characters/second with parallel file processing
- **Smart Filtering**: Automatically skips binary files, archives, and common non-text formats
- **Daemon Mode**: Run scheduled scans with configurable intervals
- **Cross-Platform**: Builds for Linux, macOS, and Windows (amd64/arm64)

## Installation

### Download Pre-built Binary

```bash
# Linux (amd64)
curl -L -o securelens-agent https://github.com/securelens/securelens-agent/releases/latest/download/securelens-agent-linux-amd64
chmod +x securelens-agent

# macOS (Apple Silicon)
curl -L -o securelens-agent https://github.com/securelens/securelens-agent/releases/latest/download/securelens-agent-darwin-arm64
chmod +x securelens-agent

# macOS (Intel)
curl -L -o securelens-agent https://github.com/securelens/securelens-agent/releases/latest/download/securelens-agent-darwin-amd64
chmod +x securelens-agent
```

### Build from Source

```bash
cd agent
make build        # Build for current platform
make build-all    # Build for all platforms
make install      # Install to /usr/local/bin
```

## Quick Start

### 1. Initialize the Agent

```bash
./securelens-agent init --api-key sl_your_api_key --api-url https://api.securelens.ai
```

This registers the agent with SecureLens and saves configuration to `~/.securelens/config.yaml`.

### 2. Run a Scan

```bash
# Scan specific directories
./securelens-agent scan /var/log /etc /home

# Scan with exclusions
./securelens-agent scan /home --exclude "*.log,*.tmp,node_modules"

# Scan without uploading results
./securelens-agent scan /var/log --upload=false
```

### 3. Run as Daemon

```bash
# Run scheduled scans every hour
./securelens-agent daemon --interval 1h --paths /var/log,/etc,/home

# Run every 30 minutes
./securelens-agent daemon --interval 30m
```

### 4. Check Status

```bash
./securelens-agent status
```

## CLI Reference

### `init`

Initialize the agent with SecureLens API credentials.

```bash
securelens-agent init --api-key <key> [--api-url <url>]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--api-key` | SecureLens API key (required) | - |
| `--api-url` | SecureLens API URL | https://api.securelens.ai |

### `scan`

Scan directories for PII/sensitive data.

```bash
securelens-agent scan <paths...> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--exclude` | Patterns to exclude | `*.gz,*.zip,*.tar` |
| `--upload` | Upload results to SecureLens | `true` |
| `--max-size` | Max file size in MB | `50` |
| `-v, --verbose` | Verbose output | `false` |

### `daemon`

Run as a daemon with scheduled scans.

```bash
securelens-agent daemon [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--interval` | Scan interval (e.g., 30m, 1h, 24h) | `1h` |
| `--paths` | Paths to scan | `/var/log,/etc,/home` |

### `status`

Show agent status and recent scan results.

```bash
securelens-agent status
```

## PII Types Detected

| Type | Severity | Description |
|------|----------|-------------|
| `EMAIL` | Medium | Email addresses |
| `PHONE` | Medium | Phone numbers (US format) |
| `SSN` | Critical | Social Security Numbers |
| `CREDIT_CARD` | Critical | Credit card numbers (with Luhn validation) |
| `IP_ADDRESS` | Low | IPv4 addresses |
| `AWS_ACCESS_KEY` | Critical | AWS access key IDs |
| `AWS_SECRET_KEY` | Critical | AWS secret access keys |
| `API_KEY` | High | Generic API keys |
| `JWT_TOKEN` | High | JSON Web Tokens |
| `PRIVATE_KEY` | Critical | RSA/EC/DSA private keys |
| `PASSPORT` | High | Passport numbers |
| `DRIVER_LICENSE` | High | Driver's license numbers |
| `IBAN` | High | International Bank Account Numbers |
| `SWIFT_CODE` | Medium | SWIFT/BIC codes |
| `GITHUB_TOKEN` | Critical | GitHub personal access tokens |
| `SLACK_TOKEN` | Critical | Slack API tokens |
| `STRIPE_KEY` | Critical | Stripe API keys |
| `DATABASE_URL` | Critical | Database connection strings |

## Output Example

```
SecureLens Agent v1.0.0
──────────────────────────────────────────────────
Scanning 3 path(s)...

[CRITICAL] /var/log/app.log:142 - EMAIL found: j***e@e****.com
[CRITICAL] /etc/config.json:15 - API_KEY found: sk-p****...****4xYz
[HIGH] /home/user/.env:3 - DATABASE_URL found: postgres://***:***@***

──────────────────────────────────────────────────
Scan complete: 1,234 files scanned, 3 findings
  Critical: 2 | High: 1 | Medium: 0 | Low: 0
  Duration: 2.3s

Uploading results to SecureLens...
✓ Results uploaded successfully!
  View results at: https://app.securelens.ai/endpoints?agent=abc123
```

## Configuration

Configuration is stored in `~/.securelens/config.yaml`:

```yaml
api_key: sl_your_api_key
api_url: https://api.securelens.ai
agent_id: abc123-def456
hostname: my-server
exclude:
  - "*.gz"
  - "*.zip"
  - "node_modules"
interval: 1h
last_scan: "2024-01-15T10:30:00Z"
```

## Running as a System Service

### Linux (systemd)

Create `/etc/systemd/system/securelens-agent.service`:

```ini
[Unit]
Description=SecureLens Device Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/securelens-agent daemon --interval 1h --paths /var/log,/etc,/home
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable securelens-agent
sudo systemctl start securelens-agent
```

### macOS (launchd)

Create `~/Library/LaunchAgents/ai.securelens.agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>ai.securelens.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/securelens-agent</string>
        <string>daemon</string>
        <string>--interval</string>
        <string>1h</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Then:

```bash
launchctl load ~/Library/LaunchAgents/ai.securelens.agent.plist
```

## Security Considerations

- The agent requires read access to directories being scanned
- API keys are stored in `~/.securelens/config.yaml` with 0600 permissions
- Sensitive values in findings are automatically masked before upload
- The agent never modifies or deletes files - it's read-only

## License

Copyright 2024 SecureLens. All rights reserved.
