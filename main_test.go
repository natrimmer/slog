package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// Mock implementations for testing

// MockFileSystem implements FileSystem interface for testing
type MockFileSystem struct {
	homeDir    string
	homeErr    error
	mkdirErr   error
	writeErr   error
	readData   []byte
	readErr    error
	readFiles  map[string][]byte // Different content for different files
	writeFiles map[string][]byte // Track what was written
	openErr    error
	openedFile *MockFile
}

type MockFile struct {
	writeErr    error
	writtenData string
	closed      bool
}

func (m *MockFile) WriteString(s string) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.writtenData += s
	return len(s), nil
}

func (m *MockFile) Close() error {
	m.closed = true
	return nil
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		writeFiles: make(map[string][]byte),
		readFiles:  make(map[string][]byte),
		openedFile: &MockFile{},
	}
}

func (m *MockFileSystem) UserHomeDir() (string, error) {
	return m.homeDir, m.homeErr
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirErr
}

func (m *MockFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writeFiles[filename] = data
	return nil
}

func (m *MockFileSystem) ReadFile(filename string) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}

	// Check if we have specific data for this file
	if data, exists := m.readFiles[filename]; exists {
		return data, nil
	}

	// Fall back to the general readData
	return m.readData, nil
}

func (m *MockFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	// We can't return a real *os.File from mock, but we'll test the logic separately
	return nil, nil
}

// MockPrinter implements Printer interface for testing
type MockPrinter struct {
	messages []string
}

func (m *MockPrinter) Print(msg string) {
	m.messages = append(m.messages, msg)
}

func (m *MockPrinter) PrintSuccess(msg string) {
	m.messages = append(m.messages, "[SUCCESS] "+msg)
}

func (m *MockPrinter) PrintError(msg string) {
	m.messages = append(m.messages, "[ERROR] "+msg)
}

func (m *MockPrinter) PrintWarning(msg string) {
	m.messages = append(m.messages, "[WARNING] "+msg)
}

func (m *MockPrinter) GetMessages() []string {
	return m.messages
}

func (m *MockPrinter) Reset() {
	m.messages = nil
}

func (m *MockPrinter) ContainsMessage(msg string) bool {
	for _, message := range m.messages {
		if strings.Contains(message, msg) {
			return true
		}
	}
	return false
}

// Test ConfigService
func TestConfigService_SaveConfig(t *testing.T) {
	tests := []struct {
		name           string
		logFile        string
		logLevels      map[string]string
		defaultLevel   string
		existingConfig *Config
		setupMock      func(*MockFileSystem)
		expectError    bool
		errorMsg       string
		expectedConfig *Config
	}{
		{
			name:         "successful save with both parameters",
			logFile:      "/tmp/test.log",
			logLevels:    map[string]string{"info": "i", "warn": "w"},
			defaultLevel: "info",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
			},
			expectError: false,
			expectedConfig: &Config{
				LogFile:      "/tmp/test.log",
				LogLevels:    map[string]string{"info": "i", "warn": "w"},
				DefaultLevel: "info",
			},
		},
		{
			name:         "update only log file",
			logFile:      "/tmp/new.log",
			logLevels:    nil,
			defaultLevel: "",
			existingConfig: &Config{
				LogFile:      "/tmp/old.log",
				LogLevels:    map[string]string{"debug": "d"},
				DefaultLevel: "debug",
			},
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/old.log", LogLevels: map[string]string{"debug": "d"}, DefaultLevel: "debug"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
			},
			expectError: false,
			expectedConfig: &Config{
				LogFile:      "/tmp/new.log",
				LogLevels:    map[string]string{"debug": "d"},
				DefaultLevel: "debug",
			},
		},
		{
			name:         "update only log levels",
			logFile:      "",
			logLevels:    map[string]string{"error": "e", "fatal": "f"},
			defaultLevel: "",
			existingConfig: &Config{
				LogFile:      "/tmp/existing.log",
				LogLevels:    map[string]string{"info": "i"},
				DefaultLevel: "info",
			},
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/existing.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
			},
			expectError: false,
			expectedConfig: &Config{
				LogFile:      "/tmp/existing.log",
				LogLevels:    map[string]string{"error": "e", "fatal": "f"},
				DefaultLevel: "info",
			},
		},
		{
			name:         "no config file provided and no existing config",
			logFile:      "",
			logLevels:    map[string]string{"info": "i"},
			defaultLevel: "",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("file not found")
			},
			expectError: false, // Should use default log file
			expectedConfig: &Config{
				LogFile:      "./log.txt",
				LogLevels:    map[string]string{"info": "i"},
				DefaultLevel: "info",
			},
		},
		{
			name:         "home directory error",
			logFile:      "/tmp/test.log",
			logLevels:    map[string]string{"info": "i"},
			defaultLevel: "info",
			setupMock: func(fs *MockFileSystem) {
				fs.homeErr = errors.New("home dir error")
			},
			expectError: true,
			errorMsg:    "error getting home directory",
		},
		{
			name:         "mkdir error",
			logFile:      "/tmp/test.log",
			logLevels:    map[string]string{"info": "i"},
			defaultLevel: "info",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.mkdirErr = errors.New("mkdir error")
			},
			expectError: true,
			errorMsg:    "error creating config directory",
		},
		{
			name:         "write file error",
			logFile:      "/tmp/test.log",
			logLevels:    map[string]string{"info": "i"},
			defaultLevel: "info",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.writeErr = errors.New("write error")
			},
			expectError: true,
			errorMsg:    "error writing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			err := configService.SaveConfig(tt.logFile, tt.logLevels, tt.defaultLevel, "")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}

				// Check that config was written correctly
				expectedPath := filepath.Join("/tmp", ".slog", "config.json")
				if data, exists := mockFS.writeFiles[expectedPath]; exists {
					var config Config
					if err := json.Unmarshal(data, &config); err != nil {
						t.Errorf("Failed to unmarshal written config: %v", err)
					} else {
						if tt.expectedConfig != nil {
							if config.LogFile != tt.expectedConfig.LogFile {
								t.Errorf("Expected log file %q, got %q", tt.expectedConfig.LogFile, config.LogFile)
							}
							if len(config.LogLevels) != len(tt.expectedConfig.LogLevels) {
								t.Errorf("Expected %d log levels, got %d", len(tt.expectedConfig.LogLevels), len(config.LogLevels))
							}
							for k, v := range tt.expectedConfig.LogLevels {
								if config.LogLevels[k] != v {
									t.Errorf("Expected log level %q:%q, got %q:%q", k, v, k, config.LogLevels[k])
								}
							}
						}
					}
				} else {
					t.Error("Config file was not written")
				}

				// Check that success message was printed
				if !mockPrinter.ContainsMessage("Configuration saved successfully") {
					t.Error("Expected success message to be printed")
				}
			}
		})
	}
}

func TestConfigService_LoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockFileSystem)
		expectErr bool
		expected  *Config
		errorMsg  string
	}{
		{
			name: "successful load",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				configJSON := `{"log_file":"/tmp/test.log","log_levels":{"info":"i","warn":"w"}}`
				fs.readData = []byte(configJSON)
			},
			expectErr: false,
			expected: &Config{
				LogFile:   "/tmp/test.log",
				LogLevels: map[string]string{"info": "i", "warn": "w"},
			},
		},
		{
			name: "home directory error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeErr = errors.New("home dir error")
			},
			expectErr: true,
			errorMsg:  "error getting home directory",
		},
		{
			name: "file read error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("file not found")
			},
			expectErr: true,
			errorMsg:  "error reading config file",
		},
		{
			name: "invalid JSON",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readData = []byte("invalid json")
			},
			expectErr: true,
			errorMsg:  "error parsing config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			config, err := configService.LoadConfig()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if config.LogFile != tt.expected.LogFile {
					t.Errorf("Expected log file %q, got %q", tt.expected.LogFile, config.LogFile)
				}
				if len(config.LogLevels) != len(tt.expected.LogLevels) {
					t.Errorf("Expected %d log levels, got %d", len(tt.expected.LogLevels), len(config.LogLevels))
				}
				for k, v := range tt.expected.LogLevels {
					if config.LogLevels[k] != v {
						t.Errorf("Expected log level %q:%q, got %q:%q", k, v, k, config.LogLevels[k])
					}
				}
			}
		})
	}
}

func TestConfigService_ViewConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockFileSystem)
		expectErr bool
		checkMsg  string
	}{
		{
			name: "successful view",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				configJSON := `{"log_file":"/tmp/test.log","log_levels":{"info":"i"}}`
				fs.readData = []byte(configJSON)
			},
			expectErr: false,
			checkMsg:  "Current Configuration:",
		},
		{
			name: "config load error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("config not found")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			err := configService.ViewConfig()

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if !mockPrinter.ContainsMessage(tt.checkMsg) {
					t.Errorf("Expected message %q to be printed", tt.checkMsg)
				}
			}
		})
	}
}

// Test LogService
func TestLogService_AppendLog(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		message       string
		setupMocks    func(*MockFileSystem)
		expectErr     bool
		errorMsg      string
		expectedLevel string
	}{
		{
			name:    "successful log with level",
			level:   "warn",
			message: "test warning message",
			setupMocks: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"warn": "w"}}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
			},
			expectErr:     false,
			expectedLevel: "WARN",
		},
		{
			name:    "successful log without level (defaults to info)",
			level:   "",
			message: "test info message",
			setupMocks: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
			},
			expectErr:     false,
			expectedLevel: "INFO",
		},
		{
			name:    "invalid UTF-8 message",
			level:   "info",
			message: string([]byte{0xff, 0xfe, 0xfd}),
			setupMocks: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
			},
			expectErr: true,
			errorMsg:  "message contains invalid UTF-8",
		},
		{
			name:    "config load error",
			level:   "info",
			message: "test message",
			setupMocks: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("config not found")
			},
			expectErr: true,
			errorMsg:  "config not found",
		},
		{
			name:    "file open error",
			level:   "info",
			message: "test message",
			setupMocks: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}}
				configJSON, _ := json.Marshal(config)
				fs.readData = configJSON
				fs.openErr = errors.New("permission denied")
			},
			expectErr: true,
			errorMsg:  "error opening log file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMocks(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			logService := NewLogService(configService, mockFS, mockPrinter)

			err := logService.AppendLog(tt.level, tt.message)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				// Note: We can't test file writing with the mock easily since os.File is not mockable
				// In a real implementation, we'd need a file writer interface
				if !mockPrinter.ContainsMessage("Logged to") {
					t.Error("Expected success message about logging")
				}
			}
		})
	}
}

// Test App integration
func TestApp_HandleConfig(t *testing.T) {
	tests := []struct {
		name           string
		logFile        string
		logLevels      map[string]string
		defaultLevel   string
		existingConfig bool
		expectErr      bool
	}{
		{
			name:         "successful config with both parameters",
			logFile:      "/tmp/test.log",
			logLevels:    map[string]string{"info": "i", "warn": "w"},
			defaultLevel: "info",
			expectErr:    false,
		},
		{
			name:           "update only levels with existing config",
			logFile:        "",
			logLevels:      map[string]string{"error": "e"},
			defaultLevel:   "",
			existingConfig: true,
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockFS.homeDir = "/tmp"
			mockPrinter := &MockPrinter{}

			if tt.existingConfig {
				config := Config{LogFile: "/tmp/existing.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				mockFS.readData = configJSON
			}

			configService := NewConfigService(mockFS, mockPrinter)
			logService := NewLogService(configService, mockFS, mockPrinter)
			app := &App{
				configService: configService,
				logService:    logService,
				printer:       mockPrinter,
			}

			err := app.HandleConfig(tt.logFile, tt.logLevels, tt.defaultLevel, "")

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestApp_HandleView(t *testing.T) {
	tests := []struct {
		name           string
		logFileContent string
		quiet          bool
		setupMock      func(*MockFileSystem)
		expectError    bool
		expectedOutput string
	}{
		{
			name:           "view log file with content",
			logFileContent: "[2024-01-15 10:30:00] INFO: Test message\n[2024-01-15 10:31:00] WARN: Warning message\n",
			quiet:          false,
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
				fs.readFiles["/tmp/test.log"] = []byte("[2024-01-15 10:30:00] INFO: Test message\n[2024-01-15 10:31:00] WARN: Warning message\n")
			},
			expectError:    false,
			expectedOutput: "Log file contents:",
		},
		{
			name:           "view log file with content (quiet mode)",
			logFileContent: "[2024-01-15 10:30:00] INFO: Test message\n[2024-01-15 10:31:00] WARN: Warning message\n",
			quiet:          true,
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
				fs.readFiles["/tmp/test.log"] = []byte("[2024-01-15 10:30:00] INFO: Test message\n[2024-01-15 10:31:00] WARN: Warning message\n")
			},
			expectError:    false,
			expectedOutput: "INFO: Test message",
		},
		{
			name:           "view empty log file",
			logFileContent: "",
			quiet:          false,
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/empty.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
				fs.readFiles["/tmp/empty.log"] = []byte("")
			},
			expectError:    false,
			expectedOutput: "Log file is empty:",
		},
		{
			name:           "view empty log file (quiet mode)",
			logFileContent: "",
			quiet:          true,
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/empty.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
				fs.readFiles["/tmp/empty.log"] = []byte("")
			},
			expectError:    false,
			expectedOutput: "", // No output expected in quiet mode for empty file
		},
		{
			name:  "log file read error",
			quiet: false,
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{LogFile: "/tmp/error.log", LogLevels: map[string]string{"info": "i"}, DefaultLevel: "info"}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
				// Don't set readFiles for error.log to trigger error
				fs.readErr = errors.New("file not found")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			logService := NewLogService(configService, mockFS, mockPrinter)
			app := &App{
				configService: configService,
				logService:    logService,
				printer:       mockPrinter,
			}

			err := app.HandleView(tt.quiet)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if tt.expectedOutput != "" {
					if !mockPrinter.ContainsMessage(tt.expectedOutput) {
						t.Errorf("Expected output containing %q", tt.expectedOutput)
					}
				} else {
					// For quiet mode empty file test, verify no header messages
					if mockPrinter.ContainsMessage("Log file is empty:") || mockPrinter.ContainsMessage("Log file contents:") {
						t.Error("Expected no header output in quiet mode for empty file")
					}
				}
			}
		})
	}
}

func TestApp_HandleConfigView(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockFileSystem)
		expectError bool
	}{
		{
			name: "view config successfully",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				config := Config{
					LogFile:      "/tmp/test.log",
					LogLevels:    map[string]string{"info": "i", "warn": "w"},
					DefaultLevel: "info",
				}
				configJSON, _ := json.Marshal(config)
				fs.readFiles["/tmp/.slog/config.json"] = configJSON
			},
			expectError: false,
		},
		{
			name: "config load error",
			setupMock: func(fs *MockFileSystem) {
				fs.homeDir = "/tmp"
				fs.readErr = errors.New("config not found")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			mockPrinter := &MockPrinter{}
			tt.setupMock(mockFS)

			configService := NewConfigService(mockFS, mockPrinter)
			logService := NewLogService(configService, mockFS, mockPrinter)
			app := &App{
				configService: configService,
				logService:    logService,
				printer:       mockPrinter,
			}

			err := app.HandleConfigView()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if !mockPrinter.ContainsMessage("Current Configuration:") {
					t.Error("Expected configuration view output")
				}
				if !mockPrinter.ContainsMessage("Configuration Usage:") {
					t.Error("Expected configuration usage output")
				}
			}
		})
	}
}

func TestApp_HandleLog(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.homeDir = "/tmp"
	config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}}
	configJSON, _ := json.Marshal(config)
	mockFS.readData = configJSON
	mockPrinter := &MockPrinter{}

	configService := NewConfigService(mockFS, mockPrinter)
	logService := NewLogService(configService, mockFS, mockPrinter)
	app := &App{
		configService: configService,
		logService:    logService,
		printer:       mockPrinter,
	}

	err := app.HandleLog("info", "test message")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

// Test version functionality
func TestApp_ShowVersion(t *testing.T) {
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

	app.ShowVersion()

	messages := mockPrinter.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected version messages, got none")
	}

	// Should contain "SLog" and version
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "SLog") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'SLog' in version output")
	}

	// Should contain the SemVer version
	foundVersion := false
	for _, msg := range messages {
		if strings.Contains(msg, "v0.0.0-dev") {
			foundVersion = true
			break
		}
	}
	if !foundVersion {
		t.Error("Expected SemVer version 'v0.0.0-dev' in version output")
	}
}

// Test help functionality
func TestApp_ShowHelp(t *testing.T) {
	mockPrinter := &MockPrinter{}
	app := &App{printer: mockPrinter}

	app.ShowHelp()

	messages := mockPrinter.GetMessages()
	if len(messages) == 0 {
		t.Error("Expected help messages, got none")
	}

	// Check for expected sections
	expectedSections := []string{
		"SLog",
		"Commands:",
		"config",
		"view",
		"help",
		"Usage:",
		"Flags:",
		"--version",
		"--help",
		"Examples:",
	}

	for _, expected := range expectedSections {
		if !mockPrinter.ContainsMessage(expected) {
			t.Errorf("Expected help to contain section: %q", expected)
		}
	}
}

// Test parseLevels function
func TestParseLevels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single level",
			input:    "info:i",
			expected: map[string]string{"info": "i"},
		},
		{
			name:     "multiple levels",
			input:    "info:i,warn:w,error:e",
			expected: map[string]string{"info": "i", "warn": "w", "error": "e"},
		},
		{
			name:     "levels with spaces",
			input:    " info : i , warn : w ",
			expected: map[string]string{"info": "i", "warn": "w"},
		},
		{
			name:     "invalid format",
			input:    "info,warn:w,invalid",
			expected: map[string]string{"warn": "w"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevels(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d levels, got %d", len(tt.expected), len(result))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Expected level %q:%q, got %q:%q", k, v, k, result[k])
				}
			}
		})
	}
}

// Test UTF-8 validation
func TestUTF8Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isValid bool
	}{
		{
			name:    "valid ASCII",
			input:   "hello world",
			isValid: true,
		},
		{
			name:    "valid UTF-8",
			input:   "hello 世界",
			isValid: true,
		},
		{
			name:    "invalid UTF-8",
			input:   string([]byte{0xff, 0xfe, 0xfd}),
			isValid: false,
		},
		{
			name:    "empty string",
			input:   "",
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utf8.ValidString(tt.input)
			if result != tt.isValid {
				t.Errorf("Expected UTF-8 validity %v for %q, got %v", tt.isValid, tt.input, result)
			}
		})
	}
}

// Test version variables
func TestVersionVariables(t *testing.T) {
	// Test that version variables are properly defined
	if version == "" {
		t.Error("version variable should not be empty")
	}

	if buildDate == "" {
		t.Error("buildDate variable should not be empty")
	}

	if commitSHA == "" {
		t.Error("commitSHA variable should not be empty")
	}

	// Test SemVer default values
	if version != "v0.0.0-dev" {
		t.Logf("Note: version is set to %q (not default 'v0.0.0-dev')", version)
	}

	if buildDate != "unknown" {
		t.Logf("Note: buildDate is set to %q (not default 'unknown')", buildDate)
	}

	if commitSHA != "unknown" {
		t.Logf("Note: commitSHA is set to %q (not default 'unknown')", commitSHA)
	}

	// Test SemVer format validation
	if !strings.HasPrefix(version, "v") {
		t.Errorf("Version should follow SemVer format and start with 'v', got: %q", version)
	}
}

// Benchmark tests
func BenchmarkParseLevels(b *testing.B) {
	input := "info:i,warn:w,error:e,debug:d,fatal:f"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parseLevels(input)
	}
}

func BenchmarkConfigService_LoadConfig(b *testing.B) {
	mockFS := NewMockFileSystem()
	mockFS.homeDir = "/tmp"
	config := Config{LogFile: "/tmp/test.log", LogLevels: map[string]string{"info": "i"}}
	configJSON, _ := json.Marshal(config)
	mockFS.readData = configJSON

	configService := NewConfigService(mockFS, &MockPrinter{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := configService.LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}
