package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/devzero-inc/oda/util"

	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

// Config config definition
type Config struct {
	// Debug persists the debug mode, so we don't have to pass it via flag, flag will override this
	Debug bool `mapstructure:"debug"`
	// ProcessInterval interval in seconds to tick and collect general information about processes - defaults to 120 seconds
	ProcessInterval int `mapstructure:"process_interval"`
	// CommandInterval interval in which to collect process information when command has been executed - defaults to 1 second
	CommandInterval int `mapstructure:"command_interval"`
	// CommandIntervalMultiplier multiplier for the command interval - defaults to 3 (cubic)
	CommandIntervalMultiplier float64 `mapstructure:"command_interval_multiplier"`
	// MaxDuration max duration that collection can run for
	MaxDuration int `mapstructure:"max_duration"`
	// MaxConcurrentCommands maximum number of concurrent commands to collect - defaults to 20
	MaxConcurrentCommands int `mapstructure:"max_concurrent_commands"`
	// RemoteCollection flag to enable remote collection - defaults to false
	RemoteCollection bool `mapstructure:"remote_collection"`
	// ServerAddress host to connect to for remote collection
	ServerAddress string `mapstructure:"server_host"`
	// SecureConnection flag to enable secure connection to the server
	SecureConnection bool `mapstructure:"secure_connection"`
	// CertFile path to the certificate file
	CertFile string `mapstructure:"cert_file"`
	// ExcludeRegex regular expression to exclude processes from collection
	ExcludeRegex string `mapstructure:"exclude_regex"`
	// ExcludeCommands regular expression to exclude commands from collection
	ExcludeCommands []string `mapstructure:"exclude_commands"`
	// ProcessCollectionType type of process collection to use, ps or psutil
	ProcessCollectionType string `mapstructure:"process_collection_type"`
	// TeamID is the team identifier for the workspace
	TeamID string `mapstructure:"team_id"`
	// UserID is the user identifier for the workspace
	UserID string `mapstructure:"user_id"`
	// UserEmail is the user identifier for the workspace
	UserEmail string `mapstructure:"user_email"`
	// WorkspaceID is the workspace identifier
	WorkspaceID string `mapstructure:"workspace_id"`
}

// SystemConfig Configuration that is not available via the configuration file
type SystemConfig struct {
	// Out is the output writer for printing information
	Out io.Writer
	// ErrOut is the error output writer for printing errors
	ErrOut io.Writer
}

// Embedding config file
//
//go:embed config.example.toml
var configExample string

// AppConfig is the global configuration instance
var AppConfig *Config

// SysConfig is the global system configuration instance
var SysConfig *SystemConfig

// SetupSysConfig initialize the system configuration instance
func SetupSysConfig() {
	SysConfig = &SystemConfig{
		// Set default output writers - currently we only use stdout and stderr, but this could potentially be changed
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

// SetupConfig initialize the configuration instance
func SetupConfig(odaDir string, user *user.User) {
	configPath := filepath.Join(odaDir, "config.toml")

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		if err := util.WriteFileAndChown(configPath, []byte(configExample), 0644, user); err != nil {
			fmt.Fprintf(SysConfig.ErrOut, "Failed to create config file: %s\n", err)
		}
	}

	viper.SetConfigFile(configPath)

	// Set default configuration values
	var config = &Config{
		Debug:                     false,
		RemoteCollection:          false,
		ProcessInterval:           3600,
		CommandInterval:           1,
		CommandIntervalMultiplier: 3,
		MaxConcurrentCommands:     20,
		ProcessCollectionType:     "ps",
		MaxDuration:               3600,
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(SysConfig.ErrOut, "Failed to read config file: %s\n", err)
	}

	if err := viper.Unmarshal(config); err != nil {
		fmt.Fprintf(SysConfig.ErrOut, "Failed to unmarshal config: %s\n", err)
	}

	AppConfig = config
}
