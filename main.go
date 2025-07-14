package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	version   = "v0.0.0-dev"
	buildDate = "unknown"
	commitSHA = "unknown"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

type Config struct {
	LogFile      string            `json:"log_file"`
	LogLevels    map[string]string `json:"log_levels"`
	DefaultLevel string            `json:"default_level"`
	WriteMode    string            `json:"write_mode"`
}

type FileSystem interface {
	UserHomeDir() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type Printer interface {
	Print(msg string)
	PrintSuccess(msg string)
	PrintError(msg string)
	PrintWarning(msg string)
}

type RealFileSystem struct{}

func (fs *RealFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFileSystem) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (fs *RealFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (fs *RealFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

type ConsolePrinter struct{}

func (p *ConsolePrinter) Print(msg string) {
	fmt.Println(msg)
}

func (p *ConsolePrinter) PrintSuccess(msg string) {
	fmt.Println(Green + msg + Reset)
}

func (p *ConsolePrinter) PrintError(msg string) {
	fmt.Println(Red + msg + Reset)
}

func (p *ConsolePrinter) PrintWarning(msg string) {
	fmt.Println(Yellow + msg + Reset)
}

type ConfigService struct {
	fs      FileSystem
	printer Printer
}

func NewConfigService(fs FileSystem, printer Printer) *ConfigService {
	return &ConfigService{fs: fs, printer: printer}
}

func (cs *ConfigService) SaveConfig(logFile string, logLevels map[string]string, defaultLevel string, writeMode string) error {
	existingConfig, _ := cs.LoadConfig()

	config := Config{
		LogFile: "./log.txt",
		LogLevels: map[string]string{
			"debug": "d",
			"info":  "i",
			"warn":  "w",
			"error": "e",
		},
		DefaultLevel: "info",
		WriteMode:    "append",
	}

	if existingConfig != nil {
		config = *existingConfig
	}

	if logFile != "" {
		config.LogFile = logFile
	}

	if len(logLevels) > 0 {
		config.LogLevels = logLevels
	}

	if defaultLevel != "" {
		config.DefaultLevel = defaultLevel
	}

	if writeMode != "" {
		if writeMode != "append" && writeMode != "prepend" {
			return fmt.Errorf("write mode must be 'append' or 'prepend'")
		}
		config.WriteMode = writeMode
	}

	if config.LogFile == "" {
		return fmt.Errorf("log file path is required")
	}

	homeDir, err := cs.fs.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".slog")
	err = cs.fs.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	err = cs.fs.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	cs.printer.PrintSuccess("Configuration saved successfully")
	cs.printer.Print(Bold + "Log File: " + Reset + config.LogFile)
	cs.printer.Print(Bold + "Log Levels: " + Reset + fmt.Sprintf("%v", config.LogLevels))
	cs.printer.Print(Bold + "Default Level: " + Reset + config.DefaultLevel)
	cs.printer.Print(Bold + "Write Mode: " + Reset + config.WriteMode)

	return nil
}

func (cs *ConfigService) LoadConfig() (*Config, error) {
	homeDir, err := cs.fs.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	configFile := filepath.Join(homeDir, ".slog", "config.json")
	data, err := cs.fs.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w\nPlease run 'config' first", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}

func (cs *ConfigService) ViewConfig() error {
	config, err := cs.LoadConfig()
	if err != nil {
		cs.printer.PrintWarning("No configuration found. Creating default configuration...")
		cs.printer.Print("")
		// Create default configuration
		err = cs.SaveConfig("", nil, "", "")
		if err != nil {
			return fmt.Errorf("error creating default configuration: %w", err)
		}
		cs.printer.Print("")
		// Load the newly created config
		config, err = cs.LoadConfig()
		if err != nil {
			return err
		}
	}

	homeDir, err := cs.fs.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %w", err)
	}
	configFile := filepath.Join(homeDir, ".slog", "config.json")

	cs.printer.Print(Bold + Cyan + "Current Configuration:" + Reset)
	cs.printer.Print(Bold + "Config File: " + Reset + configFile)
	cs.printer.Print(Bold + "Log File: " + Reset + config.LogFile)
	cs.printer.Print(Bold + "Log Levels: " + Reset + fmt.Sprintf("%v", config.LogLevels))
	cs.printer.Print(Bold + "Default Level: " + Reset + config.DefaultLevel)
	cs.printer.Print(Bold + "Write Mode: " + Reset + config.WriteMode)

	return nil
}

func (cs *ConfigService) ShowConfigUsage() {
	cs.printer.Print(Bold + Cyan + "Configuration Usage:" + Reset)
	cs.printer.Print(Bold + "Set configuration:" + Reset)
	cs.printer.Print("  slog config --file <path> --levels <level:flag,...> --default <level> --mode <append|prepend>")
	cs.printer.Print("  slog config -f <path> -l <level:flag,...> -d <level> -m <append|prepend>")
	cs.printer.Print("")
	cs.printer.Print(Bold + "Examples:" + Reset)
	cs.printer.Print("  slog config --file ./app.log --levels 'info:i,warn:w,error:e' --default info --mode append")
	cs.printer.Print("  slog config -f ./app.log -l 'debug:d,info:i' -d debug -m prepend")
	cs.printer.Print("")
	cs.printer.Print(Bold + "Flags:" + Reset)
	cs.printer.Print("  --file, -f      Path to log file")
	cs.printer.Print("  --levels, -l    Log levels in format 'level:flag,level:flag'")
	cs.printer.Print("  --default, -d   Default log level when no level flag is provided")
	cs.printer.Print("  --mode, -m      Write mode: 'append' (default) or 'prepend'")
}

type LogService struct {
	configService *ConfigService
	fs            FileSystem
	printer       Printer
}

func NewLogService(configService *ConfigService, fs FileSystem, printer Printer) *LogService {
	return &LogService{
		configService: configService,
		fs:            fs,
		printer:       printer,
	}
}

func (ls *LogService) AppendLog(level, message string) error {
	config, err := ls.configService.LoadConfig()
	if err != nil {
		return err
	}

	if !utf8.ValidString(message) {
		return fmt.Errorf("message contains invalid UTF-8")
	}

	if level == "" {
		level = config.DefaultLevel
		if level == "" {
			level = "info" // fallback if config has no default
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, strings.ToUpper(level), message)

	if config.WriteMode == "prepend" {
		// Read existing content
		existingContent, err := ls.fs.ReadFile(config.LogFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error reading existing log file: %w", err)
		}

		// Prepend new log entry
		newContent := logEntry + string(existingContent)

		// Write the combined content
		err = ls.fs.WriteFile(config.LogFile, []byte(newContent), 0644)
		if err != nil {
			return fmt.Errorf("error writing to log file: %w", err)
		}
	} else {
		// Default append mode
		file, err := ls.fs.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("error opening log file: %w", err)
		}
		defer func() {
			if file != nil {
				if err := file.Close(); err != nil {
					fmt.Printf("Warning: failed to close log file: %v\n", err)
				}
			}
		}()

		if file != nil {
			_, err = file.WriteString(logEntry)
			if err != nil {
				return fmt.Errorf("error writing to log file: %w", err)
			}
		}
	}

	ls.printer.PrintSuccess(fmt.Sprintf("Logged to %s", config.LogFile))
	return nil
}

func (ls *LogService) ViewLogFile(quiet bool) error {
	config, err := ls.configService.LoadConfig()
	if err != nil {
		return err
	}

	data, err := ls.fs.ReadFile(config.LogFile)
	if err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	if len(data) == 0 {
		if !quiet {
			ls.printer.Print(Bold + "Log file is empty: " + Reset + config.LogFile)
		}
		return nil
	}

	if !quiet {
		ls.printer.Print(Bold + "Log file contents: " + Reset + config.LogFile)
		ls.printer.Print("")
	}
	ls.printer.Print(string(data))
	return nil
}

type App struct {
	configService *ConfigService
	logService    *LogService
	printer       Printer
}

func NewApp() *App {
	fs := &RealFileSystem{}
	printer := &ConsolePrinter{}

	configService := NewConfigService(fs, printer)
	logService := NewLogService(configService, fs, printer)

	return &App{
		configService: configService,
		logService:    logService,
		printer:       printer,
	}
}

func (app *App) HandleConfig(logFile string, logLevels map[string]string, defaultLevel string, writeMode string) error {
	return app.configService.SaveConfig(logFile, logLevels, defaultLevel, writeMode)
}

func (app *App) HandleView(quiet bool) error {
	return app.logService.ViewLogFile(quiet)
}

func (app *App) HandleConfigView() error {
	err := app.configService.ViewConfig()
	if err != nil {
		return err
	}
	app.printer.Print("")
	app.configService.ShowConfigUsage()
	return nil
}

func (app *App) HandleLog(level, message string) error {
	return app.logService.AppendLog(level, message)
}

func (app *App) ShowVersion() {
	app.printer.Print(Bold + Magenta + "SLog" + Reset + " " + Dim + version + Reset)
	if version != "v0.0.0-dev" {
		app.printer.Print(Dim + "Build Date: " + buildDate + Reset)
		app.printer.Print(Dim + "Commit: " + commitSHA + Reset)
	}
	app.printer.Print(Dim + "Simple logging tool with configurable levels" + Reset)
}

func (app *App) ShowHelp() {
	app.printer.Print(Bold + Magenta + "SLog" + Reset + " " + Dim + version + Reset)
	app.printer.Print(Dim + Magenta + "Simple logging tool with configurable levels" + Reset)
	app.printer.Print("")
	app.printer.Print(Bold + "Commands:" + Reset)
	app.printer.Print("  config    Show current configuration and usage, or set new configuration")
	app.printer.Print("  view      View log file contents")
	app.printer.Print("  help      Show this help message")
	app.printer.Print("")
	app.printer.Print(Bold + "Usage:" + Reset)
	app.printer.Print("  slog [level-flag] <message>")
	app.printer.Print("")
	app.printer.Print(Bold + "Flags:" + Reset)
	app.printer.Print("  --version, -v    Show version information")
	app.printer.Print("  --help, -h       Show this help message")
	app.printer.Print("")
	app.printer.Print(Bold + "Examples:" + Reset)
	app.printer.Print("  slog config                                                    # Show current config and usage")
	app.printer.Print("  slog config --file ./app.log --levels 'info:i,warn:w,error:e' --default info --mode append")
	app.printer.Print("  slog config -f ./app.log -l 'info:i,warn:w,error:e' -d info -m prepend")
	app.printer.Print("  slog view                                                      # View log file contents")
	app.printer.Print("  slog view --quiet                                              # View log file contents without header")
	app.printer.Print("  slog \"Application started\"")
	app.printer.Print("  slog -i \"Info message\"")
	app.printer.Print("  slog -w \"Warning message\"")
}

func parseLevels(levelsStr string) map[string]string {
	levels := make(map[string]string)
	if levelsStr == "" {
		return levels
	}

	pairs := strings.Split(levelsStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) == 2 {
			levels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return levels
}

func main() {
	app := NewApp()

	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-v":
			app.ShowVersion()
			return
		case "--help", "-h":
			app.ShowHelp()
			return
		}
	}

	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	logFile := configCmd.String("file", "", "Path to log file")
	logFileShort := configCmd.String("f", "", "Path to log file (short)")
	logLevelsStr := configCmd.String("levels", "", "Log levels in format 'level:flag,level:flag' (e.g. 'info:i,warn:w,error:e')")
	logLevelsShort := configCmd.String("l", "", "Log levels in format 'level:flag,level:flag' (short)")
	defaultLevel := configCmd.String("default", "", "Default log level when no level flag is provided")
	defaultLevelShort := configCmd.String("d", "", "Default log level when no level flag is provided (short)")
	writeMode := configCmd.String("mode", "", "Write mode: 'append' (default) or 'prepend'")
	writeModeShort := configCmd.String("m", "", "Write mode: 'append' (default) or 'prepend' (short)")

	viewCmd := flag.NewFlagSet("view", flag.ExitOnError)
	quietFlag := viewCmd.Bool("quiet", false, "Don't show header, just log contents")
	quietFlagShort := viewCmd.Bool("q", false, "Don't show header, just log contents (short)")
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	if len(os.Args) < 2 {
		app.ShowHelp()
		return
	}

	var err error

	switch os.Args[1] {
	case "config":
		if len(os.Args) == 2 {
			// Show current config and usage when no arguments provided
			err = app.HandleConfigView()
			if err != nil {
				app.printer.PrintError(err.Error())
				os.Exit(1)
			}
			return
		}
		err = configCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing config arguments: %v", err))
			os.Exit(1)
		}

		// Use either long or short form
		finalLogFile := *logFile
		if finalLogFile == "" {
			finalLogFile = *logFileShort
		}

		finalLevelsStr := *logLevelsStr
		if finalLevelsStr == "" {
			finalLevelsStr = *logLevelsShort
		}

		finalDefaultLevel := *defaultLevel
		if finalDefaultLevel == "" {
			finalDefaultLevel = *defaultLevelShort
		}

		finalWriteMode := *writeMode
		if finalWriteMode == "" {
			finalWriteMode = *writeModeShort
		}

		// Check if any config parameters were provided
		if finalLogFile == "" && finalLevelsStr == "" && finalDefaultLevel == "" && finalWriteMode == "" {
			// Show current config and usage when no parameters provided
			err = app.HandleConfigView()
			if err != nil {
				app.printer.PrintError(err.Error())
				os.Exit(1)
			}
			return
		}

		var levels map[string]string
		if finalLevelsStr != "" {
			levels = parseLevels(finalLevelsStr)
		}
		err = app.HandleConfig(finalLogFile, levels, finalDefaultLevel, finalWriteMode)
	case "view":
		err = viewCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing view arguments: %v", err))
			os.Exit(1)
		}

		// Use either long or short form for quiet flag
		quiet := *quietFlag || *quietFlagShort
		err = app.HandleView(quiet)
	case "help":
		err = helpCmd.Parse(os.Args[2:])
		if err != nil {
			app.printer.PrintError(fmt.Sprintf("Error parsing help arguments: %v", err))
			os.Exit(1)
		}
		app.ShowHelp()
		return
	default:
		config, configErr := app.configService.LoadConfig()
		if configErr != nil {
			app.printer.PrintError("No configuration found. Run 'slog config' first")
			os.Exit(1)
		}

		level := ""
		message := ""

		if len(os.Args) == 2 {
			message = os.Args[1]
		} else if len(os.Args) >= 3 {
			for levelName, flagName := range config.LogLevels {
				if os.Args[1] == "-"+flagName || os.Args[1] == "--"+levelName {
					level = levelName
					message = strings.Join(os.Args[2:], " ")
					break
				}
			}
			if level == "" {
				message = strings.Join(os.Args[1:], " ")
			}
		}

		if message == "" {
			app.printer.PrintError("No message provided")
			os.Exit(1)
		}

		err = app.HandleLog(level, message)
	}

	if err != nil {
		app.printer.PrintError(err.Error())
		os.Exit(1)
	}
}
