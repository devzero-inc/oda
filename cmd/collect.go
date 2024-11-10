package cmd

import (
	"time"

	"github.com/devzero-inc/oda/client"
	"github.com/devzero-inc/oda/collector"
	"github.com/devzero-inc/oda/config"
	"github.com/devzero-inc/oda/logging"
	"github.com/devzero-inc/oda/process"
	"github.com/devzero-inc/oda/user"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// newCollectCmd creates a new collect command.
func newCollectCmd() *cobra.Command {
	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect command and system information",
		Long:  `Collect and process command and system information.`,
		RunE:  collect,
	}

	collectCmd.Flags().BoolP("auto-credentials", "a", false, "Try to automatically generate the credentails")
	collectCmd.Flags().BoolP("workspace", "w", false, "Is collection executed in a DevZero workspace")

	return collectCmd
}

func collect(cmd *cobra.Command, _ []string) error {
	setupConfig()

	autoCredentials, err := cmd.Flags().GetBool("auto-credentials")
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to get auto-credentials flag")
		return errors.Wrap(err, "failed to get auto-credentials flag")
	}

	isWorkspace, err := cmd.Flags().GetBool("workspace")
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to get workspace flag")
		return errors.Wrap(err, "failed to get workspace flag")
	}

	user.Conf, err = user.GetConfig()
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to get os config")
		return errors.Wrap(err, "failed to get os config, please run 'oda install' first")
	}

	var grpcClient *client.Client
	if config.AppConfig.RemoteCollection {
		logging.Log.Info().Msg("Remote collection is enabled")
		grpcConfig := client.Config{
			Address:          config.AppConfig.ServerAddress,
			SecureConnection: config.AppConfig.SecureConnection,
			CertFile:         config.AppConfig.CertFile,
			Timeout:          60,
		}
		grpcClient, err = client.NewClient(grpcConfig)
		if err != nil {
			logging.Log.Error().Err(err).Msg("Failed to create client")
			return errors.Wrap(err, "failed to create client")
		}
	}

	intervalConfig := collector.IntervalConfig{
		ProcessInterval:           time.Duration(config.AppConfig.ProcessInterval) * time.Second,
		CommandInterval:           time.Duration(config.AppConfig.CommandInterval) * time.Second,
		CommandIntervalMultiplier: config.AppConfig.CommandIntervalMultiplier,
		MaxConcurrentCommands:     config.AppConfig.MaxConcurrentCommands,
		MaxDuration:               time.Duration(config.AppConfig.MaxDuration) * time.Second,
	}

	procCol, err := process.NewFactory(logging.Log).Create(config.AppConfig.ProcessCollectionType)
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to create process collector")
		return errors.Wrap(err, "failed to create process collector")
	}

	auth := collector.AuthConfig{
		UserID:      config.AppConfig.UserID,
		TeamID:      config.AppConfig.TeamID,
		WorkspaceID: config.AppConfig.WorkspaceID,
		UserEmail:   config.AppConfig.UserEmail,
	}

	if autoCredentials {
		logging.Log.Debug().Msg("Auto-credentials flag is set to true")
		if isWorkspace {
			logging.Log.Debug().Msg("Workspace flag is set to true")
			auth, err = user.ReadDZWorkspaceConfig()
			if err != nil {
				return errors.Wrap(err, "failed to read DevZero config")
			}
		} else {
			logging.Log.Debug().Msg("Workspace flag is set to false")
			path, err := user.GetStoragePath(config.OSType(user.Conf.Os), user.Conf.HomeDir)
			logging.Log.Debug().Msgf("Storage path: %s", path)
			if err != nil {
				logging.Log.Error().Err(err).Msg("Failed to get storage path")
				return errors.Wrap(err, "failed to get storage path")
			}
			auth, err = user.ReadDZCliConfig(path)
			if err != nil {
				return errors.Wrap(err, "failed to read DevZero config")
			}
		}
		logging.Log.Debug().Msgf("Auth: %+v", auth)
	}

	collectorInstance := collector.NewCollector(
		collector.SocketPath,
		grpcClient,
		logging.Log,
		intervalConfig,
		auth,
		config.AppConfig.ExcludeRegex,
		procCol,
	)

	collectorInstance.Collect()

	return nil
}
