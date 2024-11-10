package daemon

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/devzero-inc/oda/config"
	"github.com/devzero-inc/oda/util"

	"github.com/spf13/afero"

	"github.com/rs/zerolog"
)

const (
	PlistFilePath               = "Library/LaunchAgents"
	PlistSudoFilePath           = "/Library/LaunchDaemons"
	PlistName                   = "oda.plist"
	UserServicedFilePath        = ".config/systemd/user"
	RootServicedFilePath        = "/etc/systemd/system"
	ServicedName                = "oda.service"
	LinuxDaemonTemplateLocation = "services/oda.service"
	MacOSDaemonTemplateLocation = "services/oda.plist"
	ServicePermission           = 0644
	DirPermission               = 0755
	BaseCollectCommand          = "collect"
)

// Embedding scripts directory
//
//go:embed services/*
var templateFS embed.FS

// Config is the configuration for the daemon service
type Config struct {
	ExePath             string
	HomeDir             string
	Os                  config.OSType
	IsRoot              bool
	SudoExecUser        *user.User
	AutoCredential      bool
	IsWorkspace         bool
	ShellTypeToLocation map[config.ShellType]string
	BaseCommandPath     string
}

// Daemon is the service that configures background service
type Daemon struct {
	config *Config
	logger zerolog.Logger
}

// NewDaemon creates a new daemon service
func NewDaemon(conf *Config, logger zerolog.Logger) *Daemon {
	return &Daemon{
		config: conf,
		logger: logger,
	}
}

// InstallDaemonConfiguration installs the daemon service configuration
func (d *Daemon) InstallDaemonConfiguration() error {
	d.logger.Info().Msg("Installing daemon service...")

	filePath, templatePath, err := buildConfigurationPath(d.config.Os, d.config.IsRoot, d.config.HomeDir)
	if err != nil {
		return err
	}

	tmpl, err := template.ParseFS(templateFS, templatePath)
	if err != nil {
		d.logger.Err(err).Msg("Failed to parse config template")
		return err
	}

	var content bytes.Buffer
	var tmpConf = map[string]interface{}{
		"BinaryPath": d.config.ExePath,
		"Home":       d.config.HomeDir,
	}

	if d.config.SudoExecUser != nil {
		tmpConf["Username"] = d.config.SudoExecUser.Username

		group, err := user.LookupGroupId(d.config.SudoExecUser.Gid)
		if err != nil {
			return err
		}
		tmpConf["Group"] = group.Name
	}

	d.logger.Debug().Msgf("Base command path: %s", d.config.BaseCommandPath)
	commands := strings.Split(d.config.BaseCommandPath, " ")

	collectCmd := []string{}

	// TODO: fix this thing, as it is absolutly not robust, and it might not work in all cases
	if len(commands) > 0 && commands[0] != "oda" {
		// Extract the executable name from ExePath
		executableName := filepath.Base(d.config.ExePath)

		// Compare and adjust the commands slice
		if executableName == commands[0] {
			commands = commands[1:]
		}

		for _, command := range commands {
			d.logger.Debug().Msgf("Checking command path: %s", command)
			collectCmd = append(collectCmd, command)
			if command == "oda" {
				break
			}
		}

	}

	collectCmd = append(collectCmd, BaseCollectCommand)

	d.logger.Info().Msgf("Base path: %v", collectCmd)

	if d.config.AutoCredential {
		collectCmd = append(collectCmd, "-a")
	}
	if d.config.IsWorkspace {
		collectCmd = append(collectCmd, "-w")
	}

	// create command from args in colllectCmd
	var command string
	for _, arg := range collectCmd {
		command += arg + " "
	}

	tmpConf["CollectCommand"] = command
	tmpConf["CollectCommandSplit"] = collectCmd

	if err := tmpl.Execute(&content, tmpConf); err != nil {
		d.logger.Err(err).Msg("Failed to execute daemon template")
		return err
	}

	if err := util.Fs.MkdirAll(filepath.Dir(filePath), DirPermission); err != nil {
		d.logger.Err(err).Msg("Failed to create directories for daemon files")
		return fmt.Errorf("failed to create directories for daemon files: %w", err)
	}

	if err := afero.WriteFile(util.Fs, filePath, content.Bytes(), ServicePermission); err != nil {
		d.logger.Err(err).Msg("Failed to write daemon files")
		return fmt.Errorf("failed to write daemon files: %w", err)
	}

	d.logger.Info().Msg("Daemon service installed successfully")

	return nil
}

// DestroyDaemonConfiguration removes the daemon service configuration
func (d *Daemon) DestroyDaemonConfiguration() error {
	d.logger.Info().Msg("Uninstalling daemon service...")

	filePath, _, err := buildConfigurationPath(d.config.Os, d.config.IsRoot, d.config.HomeDir)
	if err != nil {
		return err
	}

	if err := util.Fs.Remove(filePath); err != nil {
		d.logger.Err(err).Msg("Failed to remove daemon service file")
		return err
	}

	d.logger.Info().Msg("Daemon service file removed successfully")

	return nil
}

// StartDaemon starts the daemon service
func (d *Daemon) StartDaemon() error {
	d.logger.Info().Msg("Starting daemon service...")

	switch d.config.Os {
	case config.Linux:
		if err := startLinuxDaemon(d.config.IsRoot); err != nil {
			d.logger.Err(err).Msg("Failed to start daemon service")
			return err
		}
	case config.MacOS:
		if err := startMacOSDaemon(d.config.HomeDir, d.config.IsRoot); err != nil {
			d.logger.Err(err).Msg("Failed to start daemon service")
			return err
		}
	default:
		d.logger.Error().Msg("Unsupported operating system")
		return fmt.Errorf("unsupported operating system")
	}

	d.logger.Info().Msg("Daemon service started successfully")

	return nil
}

// StopDaemon stops the daemon service
func (d *Daemon) StopDaemon() error {
	d.logger.Info().Msg("Stopping daemon service")

	switch d.config.Os {
	case config.Linux:
		if err := stopLinuxDaemon(d.config.IsRoot); err != nil {
			d.logger.Err(err).Msg("Failed to stop daemon service")
			return err
		}
	case config.MacOS:
		if err := stopMacOSDaemon(d.config.HomeDir, d.config.IsRoot); err != nil {
			d.logger.Err(err).Msg("Failed to stop daemon service")
			return err
		}
	default:
		d.logger.Error().Msg("Unsupported operating system")
		return fmt.Errorf("unsupported operating system")
	}

	d.logger.Info().Msg("Daemon service stopped successfully")

	return nil
}

// ReloadDaemon signals the daemon to reload its configuration.
func (d *Daemon) ReloadDaemon() error {
	d.logger.Info().Msg("Reloading daemon service...")

	switch d.config.Os {
	case config.Linux:
		return reloadLinuxDaemon(d.config.IsRoot)
	case config.MacOS:
		return reloadMacOSDaemon(d.config.HomeDir, d.config.IsRoot)
	default:
		d.logger.Error().Msg("Unsupported operating system for reload")
		return fmt.Errorf("unsupported operating system")
	}

}

// reloadLinuxDaemon reloads the daemon service on Linux using systemctl.
func reloadLinuxDaemon(isRoot bool) error {
	cmd := exec.Command("systemctl", "--user", "reload", ServicedName)
	if isRoot {
		cmd = exec.Command("systemctl", "reload", ServicedName)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload daemon service: %v", stderr.String())
	}

	return nil
}

// reloadMacOSDaemon reloads the daemon service on macOS
func reloadMacOSDaemon(homeDir string, isRoot bool) error {
	stopErr := stopMacOSDaemon(homeDir, isRoot)
	if stopErr != nil {
		return stopErr
	}
	startErr := startMacOSDaemon(homeDir, isRoot)
	if startErr != nil {
		return startErr
	}

	return nil
}

// startLinuxDaemon starts the daemon service on Linux
func startLinuxDaemon(isRoot bool) error {
	if !checkLogindService() && !isRoot {
		return fmt.Errorf("logind service is not available, and you need to be root to enable the daemon service, or enable logind service manually")
	}

	enableCmd := exec.Command("systemctl", "--user", "enable", ServicedName)
	if isRoot {
		enableCmd = exec.Command("systemctl", "enable", ServicedName)
	}
	var stderr bytes.Buffer
	enableCmd.Stderr = &stderr

	if err := enableCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable daemon service: %v", stderr.String())
	}

	cmd := exec.Command("systemctl", "--user", "start", ServicedName)
	if isRoot {
		cmd = exec.Command("systemctl", "start", ServicedName)
	}
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start daemon service: %v", stderr.String())
	}

	return nil
}

// startMacOSDaemon starts the daemon service on macOS
func startMacOSDaemon(homeDir string, isRoot bool) error {
	servicePath := filepath.Join(homeDir, PlistFilePath)
	if isRoot {
		servicePath = PlistSudoFilePath
	}
	path := filepath.Join(servicePath, PlistName)
	cmd := exec.Command("launchctl", "load", "-w", path)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start daemon service: %v", stderr.String())
	}

	return nil
}

// stopLinuxDaemon stops the daemon service on Linux
func stopLinuxDaemon(isRoot bool) error {
	cmd := exec.Command("systemctl", "--user", "stop", ServicedName)
	if isRoot {
		cmd = exec.Command("systemctl", "stop", ServicedName)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop daemon service: %v", stderr.String())
	}

	return nil
}

// stopMacOSDaemon stops the daemon service on macOS
func stopMacOSDaemon(homeDir string, isRoot bool) error {
	servicePath := filepath.Join(homeDir, PlistFilePath)
	if isRoot {
		servicePath = PlistSudoFilePath
	}
	path := filepath.Join(servicePath, PlistName)
	cmd := exec.Command("launchctl", "unload", "-w", path)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop daemon service: %v", stderr.String())
	}

	return nil
}

// buildConfigurationPath builds the path to the daemon service configuration file, and the template
func buildConfigurationPath(os config.OSType, isRoot bool, homeDir string) (string, string, error) {
	var filePath string
	var templateLocation string

	switch os {
	case config.Linux:
		servicePath := filepath.Join(homeDir, UserServicedFilePath)

		if !checkLogindService() && !isRoot {
			return "", "", fmt.Errorf("logind service is not available")
		}

		if isRoot {
			servicePath = RootServicedFilePath
		}

		filePath = filepath.Join(servicePath, ServicedName)
		templateLocation = LinuxDaemonTemplateLocation
	case config.MacOS:
		servicePath := filepath.Join(homeDir, PlistFilePath)

		if isRoot {
			servicePath = PlistSudoFilePath
		}

		filePath = filepath.Join(servicePath, PlistName)
		templateLocation = MacOSDaemonTemplateLocation
	default:
		return "", "", fmt.Errorf("unsupported operating system")
	}

	return filePath, templateLocation, nil
}

// Check if logind service is available on the system, because there are Linux systems where
// users deliberately disable logind service, and in such cases, the daemon service will not work
// on a user level, and we have to force sudo usage
func checkLogindService() bool {
	var stderr bytes.Buffer
	cmd := exec.Command("systemctl", "is-enabled", "systemd-logind.service")
	cmd.Stderr = &stderr
	err := cmd.Run()
	logindStatus := stderr.String()

	if err != nil || logindStatus == "masked" || logindStatus == "disabled" {
		return false
	}

	return true
}
