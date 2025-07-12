# slog

A simple CLI tool for structured logging with configurable levels and file output.

## Installation

### Option 1: Using Go

```bash
go install github.com/natrimmer/slog@latest
```

### Option 2: Build from source

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

# View current configuration
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
# Configure log file and levels (long form)
slog config --file /path/to/logfile.log --levels "debug:d,info:i,warn:w,error:e"

# Configure log file and levels (short form)
slog config -f /path/to/logfile.log -l "debug:d,info:i,warn:w,error:e"

# Configure only log file (keeps existing levels)
slog config --file /var/log/app.log

# Configure only levels (keeps existing file)
slog config --levels "info:i,error:e"

# View current configuration
slog view
```

### Logging

```bash
# Log message with default level (info)
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

Supported levels (case-insensitive):

- `debug` - Detailed diagnostic information
- `info` - General information messages
- `warn` - Warning messages
- `error` - Error messages

## Example Usage

### Initial Setup

```bash
$ slog config --file /var/log/myapp.log --levels "info:i,warn:w,error:e"
Configuration saved successfully.

$ slog view
Current configuration:
Log File: /var/log/myapp.log
Log Levels: info, warn, error
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

## Output Format

Logs are written in JSON format with the following fields:

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "info",
  "message": "Application started successfully"
}
```

## Configuration Storage

Configuration is stored in `~/.slog/config.json`:

```json
{
  "logFile": "/var/log/myapp.log",
  "logLevels": {
    "info": "i",
    "warn": "w", 
    "error": "e"
  }
}
```

## Features

- Configurable log levels with filtering
- JSON structured output format
- Persistent configuration storage
- UTF-8 message validation
- Cross-platform file handling
- Clean error handling and user feedback

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
