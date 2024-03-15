package daemon

import (
	"bytes"
	"embed"
	"html/template"
	"lda/config"
	"lda/logging"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	DaemonPlistFilePath    = "Library/LaunchAgents"
	DaemonPlistName        = "lda.plist"
	DaemonServicedFilePath = ".config/systemd/user"
	DaemonServicedName     = "lda.service"
	DaemonPermission       = 0644

	dirPermission = 0755
)

// Embedding scripts directory
//go:embed configs/*
var templateFS embed.FS

func InitDaemonConfiguration() {

	logging.Log.Info().Msg("Installing daemon service")

	var filePath string
	var configLocation string

	if config.OS == config.Linux {
		filePath = filepath.Join(
			config.HomeDir,
			DaemonServicedFilePath,
			DaemonServicedName)
		configLocation = "configs/lda.service"
	} else if config.OS == config.MacOS {
		filePath = filepath.Join(
			config.HomeDir,
			DaemonPlistFilePath,
			DaemonPlistName)
		configLocation = "configs/lda.plist"
	}

	if filePath == "" {
		logging.Log.Error().Msg("Unsupported operating system")
		return
	}

	// Check if the file already exists
	if _, err := os.Stat(filePath); err == nil {
		logging.Log.Info().Msg("Daemon service file already exists")
		return
	} else if !os.IsNotExist(err) {
		// An error other than "not exists", e.g., permission issues
		logging.Log.Err(err).Msg("Failed to check daemon service file")
		return
	}

	shellTempl, err := template.ParseFS(templateFS, configLocation)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to parse config template")
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	var content bytes.Buffer
	if err := shellTempl.Execute(&content, map[string]interface{}{
		"BinaryPath": exePath,
	}); err != nil {
		logging.Log.Err(err).Msg("Failed to execute daemon template")
		return
	}

	// Extract the directory part of the file path
	dirPath := filepath.Dir(filePath)

	// Create all directories in the path if they don't exist
	if err := os.MkdirAll(dirPath, dirPermission); err != nil {
		logging.Log.Err(err).Msg("Failed to create directories for daemon files")
		return
	}

	if err := os.WriteFile(filePath, content.Bytes(), DaemonPermission); err != nil {
		logging.Log.Err(err).Msg("Failed to write daemon files")
		return
	}

	logging.Log.Info().Msg("Daemon service installed successfully")
}

func DestroyDaemonConfiguration() {

	logging.Log.Info().Msg("Uninstalling daemon service")

	var filePath string

	if config.OS == config.Linux {
		filePath = filepath.Join(
			config.HomeDir,
			DaemonServicedFilePath,
			DaemonServicedName)
	} else if config.OS == config.MacOS {
		filePath = filepath.Join(
			config.HomeDir,
			DaemonPlistFilePath,
			DaemonPlistName)
	}

	if filePath == "" {
		logging.Log.Error().Msg("Unsupported operating system")
		return
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logging.Log.Info().Msg("Daemon service file does not exist")
		return
	} else if err != nil {
		// An error other than "not exists", e.g., permission issues
		logging.Log.Err(err).Msg("Failed to check daemon service file")
		return
	}

	// File exists, proceed with removal
	err := os.Remove(filePath)
	if err != nil {
		logging.Log.Err(err).Msg("Failed to remove daemon service file")
		return
	}

	logging.Log.Info().Msg("Daemon service file removed successfully")
}

func StartDeamon() {

	logging.Log.Info().Msg("Starting daemon service")

	var cmd *exec.Cmd

	if config.OS == config.Linux {
		cmd = exec.Command(
			"systemctl",
			"--user",
			"start",
			DaemonServicedName)
	} else if config.OS == config.MacOS {
		path := filepath.Join(
			config.HomeDir,
			DaemonPlistFilePath,
			DaemonPlistName)
		cmd = exec.Command(
			"launchctl",
			"load",
			"-w",
			path)
	}

	if err := cmd.Run(); err != nil {
		logging.Log.Err(err).Msg("Failed to start daemon service")
		return
	}

	logging.Log.Info().Msg("Daemon service started successfully")
}

func StopDaemon() {

	logging.Log.Info().Msg("Stoping daemon service")

	var cmd *exec.Cmd

	if config.OS == config.Linux {
		cmd = exec.Command(
			"systemctl",
			"--user",
			"stop",
			DaemonServicedName)
	} else if config.OS == config.MacOS {
		path := filepath.Join(
			config.HomeDir,
			DaemonPlistFilePath,
			DaemonPlistName)
		cmd = exec.Command(
			"launchctl",
			"unload",
			"-w",
			path)
	}

	if err := cmd.Run(); err != nil {
		logging.Log.Err(err).Msg("Failed to stop daemon service")
		return
	}

	logging.Log.Info().Msg("Daemon service stoped successfully")
}