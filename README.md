# slog

A simple CLI tool for structured logging with configurable levels and file output.

## Features

- Configurable log levels with short flag mapping
- Configurable default log level for messages without explicit level
- Plain text structured output format
- Persistent configuration storage
- UTF-8 message validation
- Cross-platform file handling

## Installation

### Option 1: Download binary

Download the pre-built binary for your platform from the [GitHub Releases page](https://github.com/natrimmer/slog/releases/latest):

```bash
# Example for Linux (amd64)
curl -L https://github.com/natrimmer/slog/releases/latest/download/slog_linux_amd64 -o slog
chmod +x slog
sudo mv slog /usr/local/bin/

# Example for macOS (intel)
curl -L https://github.com/natrimmer/slog/releases/latest/download/slog_darwin_amd64 -o slog
chmod +x slog
sudo mv slog /usr/local/bin/

# Example for macOS (Apple Silicon)
curl -L https://github.com/natrimmer/slog/releases/latest/download/slog_darwin_arm64 -o slog
chmod +x slog
sudo mv slog /usr/local/bin/
```

### Option 2: Using Go

```bash
go install github.com/natrimmer/slog@latest
```

### Option 3: Build from source

```bash
git clone https://github.com/natrimmer/slog.git
cd slog
go build
```

## Quick Start

```bash
# Get help
slog

# Configure logging
slog config --file /var/log/app.log --levels "info:i,warn:w,error:e"

# Log a message
slog "Application started"

# Log with specific level
slog -e "Database connection failed"

# View current configuration and usage
slog config

# View log file contents
slog view
```

## Commands

### Help and Version

```bash
slog              # Show help
slog --help       # Show help
slog --version    # Show version info
```

### Configuration

```bash
# Show current configuration and usage
slog config

# Configure log file, levels, and default level (long form)
slog config --file /path/to/logfile.log --levels "debug:d,info:i,warn:w,error:e" --default info

# Configure log file, levels, and default level (short form)
slog config -f /path/to/logfile.log -l "debug:d,info:i,warn:w,error:e" -d info

# Configure only log file (keeps existing levels)
slog config --file /var/log/app.log

# Configure only levels (keeps existing file and default)
slog config --levels "info:i,error:e"

# Configure only default level (keeps existing file and levels)
slog config --default warn
```

### Viewing

```bash
# View log file contents
slog view

# View log file contents without header (quiet mode)
slog view --quiet
slog view -q
```

### Logging

```bash
# Log message with configured default level
slog "Your message here"

# Log with specific level using short flags
slog -d "Debug information"
slog -i "Info message"
slog -w "Warning message"
slog -e "Error occurred"

# Log with specific level using long flags
slog --debug "Debug information"
slog --info "Info message"
slog --warn "Warning message"
slog --error "Error occurred"
```

## Log Levels

Log levels are configured via the `--levels` flag using the format `level:flag`. Any level name can be used, but common ones include:

- `debug:d` - Detailed diagnostic information
- `info:i` - General information messages
- `warn:w` - Warning messages
- `error:e` - Error messages

Note: The tool does not filter by configured levels - it accepts any level for logging.

## Example Usage

### Initial Setup

```bash
$ slog config --file /var/log/myapp.log --levels "info:i,warn:w,error:e" --default info
Configuration saved successfully.

$ slog config
Current Configuration:
Log File: /var/log/myapp.log
Log Levels: map[error:e info:i warn:w]
Default Level: info

Configuration Usage:
Set configuration:
  slog config --file <path> --levels <level:flag,...> --default <level>
  slog config -f <path> -l <level:flag,...> -d <level>
```

### Logging Messages

```bash
$ slog "Application started successfully"
Logged to /var/log/myapp.log

$ slog -w "Database connection timeout"
Logged to /var/log/myapp.log

$ slog --error "Failed to process request"
Logged to /var/log/myapp.log
```

### Viewing Log File Contents

```bash
$ slog view
Log file contents: /var/log/myapp.log

[2024-01-15 10:30:00] INFO: Application started successfully
[2024-01-15 10:31:00] WARN: Database connection timeout
[2024-01-15 10:32:00] ERROR: Failed to process request

$ slog view --quiet
[2024-01-15 10:30:00] INFO: Application started successfully
[2024-01-15 10:31:00] WARN: Database connection timeout
[2024-01-15 10:32:00] ERROR: Failed to process request
```

## Output Format

Logs are written in plain text format:

```
[2006-01-02 15:04:05] INFO: Application started successfully
[2006-01-02 15:04:05] WARN: Database connection timeout
[2006-01-02 15:04:05] ERROR: Failed to process request
```

## Configuration Storage

Configuration is stored in `~/.slog/config.json`:

```json
{
  "log_file": "/var/log/myapp.log",
  "log_levels": {
    "debug": "d",
    "info": "i",
    "warn": "w",
    "error": "e"
  },
  "default_level": "info"
}
```

## Development

### Building from Source

```bash
git clone https://github.com/natrimmer/slog.git
cd slog

# Run tests
go test

# Run tests with coverage
go test -cover

# Build
go build

# Run linter (if available)
golangci-lint run
```
