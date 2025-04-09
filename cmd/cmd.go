package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/devzero-inc/oda/config"
	"github.com/devzero-inc/oda/daemon"
	"github.com/devzero-inc/oda/database"
	"github.com/devzero-inc/oda/job"
	"github.com/devzero-inc/oda/logging"
	"github.com/devzero-inc/oda/resources"
	"github.com/devzero-inc/oda/shell"
	"github.com/devzero-inc/oda/user"
	"github.com/devzero-inc/oda/util"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Verbose Define verbose flag variables
var Verbose bool

var installFlags struct {
	shells         []string
	nonInteractive bool
}

// newInstallCmd creates a new install command
func newInstallCmd() *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install daemon runner",
		Long:  `Install daemon runner for ODA Project.`,
		RunE:  install,
	}

	installCmd.Flags().StringSliceVarP(&installFlags.shells, "shell", "s", []string{}, fmt.Sprintf("Shells to instrument %+v; --shell=all for all shells", config.SupportedShells))
	installCmd.Flags().BoolVarP(&installFlags.nonInteractive, "non-interactive", "n", false, "Run installation in non-interactive mode")
	installCmd.Flags().BoolP("auto-credentials", "a", false, "Try to automatically generate the credentails")
	installCmd.Flags().BoolP("workspace", "w", false, "Is collection executed in a DevZero workspace")

	return installCmd
}

func NewOdaCmd() *cobra.Command {
	odaCmd := &cobra.Command{
		Use:   "oda",
		Short: "Command line manager for ODA project.",
		Long: `Command line manager for ODA Project.
        Complete documentation is available at https://devzero.io`,
		Run: oda,
	}

	odaCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Verbosity")

	odaCmd.AddCommand(
		newVersionCmd(),
		newCollectCmd(),
		newStartCmd(),
		newStopCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newServeCmd(),
		newReloadCmd(),
		newConfigCmd(),
	)

	return odaCmd
}

// newUninstallCmd creates a new uninstall command
func newUninstallCmd() *cobra.Command {
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall daemon runner",
		Long:  `Uninstall daemon runner for ODA Project.`,
		RunE:  uninstall,
	}

	return uninstallCmd
}

// newStartCmd creates a new start command
func newStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start daemon runner",
		Long:  `Start daemon runner for ODA Project.`,
		RunE:  start,
	}

	return startCmd
}

// newStopCmd creates a new stop command
func newStopCmd() *cobra.Command {
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop daemon runner",
		Long:  `Stop daemon runner for ODA Project.`,
		RunE:  stop,
	}

	return stopCmd
}

// newServeCmd creates a new serve command
func newServeCmd() *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve local client",
		Long:  `Serve local frontend client for ODA Project.`,
		RunE:  serve,
	}

	serveCmd.Flags().StringP("port", "p", "8080", "Port to serve the frontend client")

	return serveCmd
}

// newConfigCmd creates a new config command
func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Print current configuration",
		Long:  `Display current configuration for ODA Project.`,
		RunE:  displayConfig,
	}

	return configCmd
}

// newReloadCmd creates a new reload command
func newReloadCmd() *cobra.Command {
	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload daemon runner",
		Long:  `Reload daemon runner for ODA Project.`,
		RunE:  reload,
	}

	return reloadCmd
}

const (
	// days are amount of days that old data will be retained
	days = 5
	// hours are amount of hours that ticker in job will use to run cleanup job
	hours = 24
)

func setupConfig() {
	// setting up the system configuration
	config.SetupSysConfig()

	// Setup afero FS layer
	util.SetupFS()

	sudoExecUser, isRoot, err := config.GetUserConfig()
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to get user configuration: %s\n", err)
		os.Exit(1)
	}
	osConf, osName, err := config.GetOS()
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to get OS: %s\n", err)
		os.Exit(1)
	}
	homeDir, err := config.GetHomeDir(isRoot, sudoExecUser)
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to get home directory: %s\n", err)
		os.Exit(1)
	}
	odaDir, err := config.GetOdaDir(homeDir, sudoExecUser)
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to get ODA directory: %s\n", err)
		os.Exit(1)
	}
	exePath, err := config.GetOdaBinaryPath()
	if err != nil {
		fmt.Fprintf(config.SysConfig.ErrOut, "Failed to get executable path: %s\n", err)
		os.Exit(1)
	}

	// setting up optional application configuration
	config.SetupConfig(odaDir, sudoExecUser)

	// setup database and run migrations
	database.Setup(odaDir, sudoExecUser)
	database.RunMigrations()

	// setting up the Logger
	// TODO: consider adding verbose levels
	if config.AppConfig.Debug || Verbose {
		logging.Setup(os.Stdout, true)
	} else {
		logging.Setup(io.Discard, false)
	}

	// Configure default user globals
	user.Conf = &user.Config{
		Os:      int64(osConf),
		OsName:  osName,
		HomeDir: homeDir,
		OdaDir:  odaDir,
		IsRoot:  isRoot,
		ExePath: exePath,
		User:    sudoExecUser,
	}

	// run cleanup job
	job.Cleanup(hours, days)
}

// Execute is the entry point for the command line
func Execute() {
	odaCmd := NewOdaCmd()
	if err := odaCmd.Execute(); err != nil {
		logging.Log.Err(err).Msg("Failed to execute main oda command")
		os.Exit(1)
	}
}

func oda(cmd *cobra.Command, _ []string) {
	if err := cmd.Help(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to show help")
	}
}

func reload(_ *cobra.Command, _ []string) error {

	setupConfig()

	user.ConfigureUserSystemInfo(user.Conf)

	daemonConf := &daemon.Config{
		ExePath:             user.Conf.ExePath,
		HomeDir:             user.Conf.HomeDir,
		IsRoot:              user.Conf.IsRoot,
		Os:                  config.OSType(user.Conf.Os),
		SudoExecUser:        user.Conf.User,
		ShellTypeToLocation: user.Conf.ShellTypeToLocation,
	}
	dmn := daemon.NewDaemon(daemonConf, logging.Log)

	fmt.Fprintln(config.SysConfig.Out, "Reloading ODA daemon...")
	if err := dmn.ReloadDaemon(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to reload daemon")
		return errors.Wrap(err, "failed to reload ODA daemon")
	}
	fmt.Fprintln(config.SysConfig.Out, "Reloading ODA daemon finished.")
	return nil
}

func start(_ *cobra.Command, _ []string) error {

	setupConfig()

	user.ConfigureUserSystemInfo(user.Conf)

	daemonConf := &daemon.Config{
		ExePath:             user.Conf.ExePath,
		HomeDir:             user.Conf.HomeDir,
		IsRoot:              user.Conf.IsRoot,
		Os:                  config.OSType(user.Conf.Os),
		SudoExecUser:        user.Conf.User,
		ShellTypeToLocation: user.Conf.ShellTypeToLocation,
	}
	dmn := daemon.NewDaemon(daemonConf, logging.Log)

	fmt.Fprintln(config.SysConfig.Out, "Starting ODA daemon...")
	if err := dmn.StartDaemon(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to start daemon")
		return errors.Wrap(err, "failed to start ODA daemon")
	}
	fmt.Fprintln(config.SysConfig.Out, "ODA daemon started.")
	return nil
}

func stop(_ *cobra.Command, _ []string) error {

	setupConfig()

	user.ConfigureUserSystemInfo(user.Conf)

	daemonConf := &daemon.Config{
		ExePath:             user.Conf.ExePath,
		HomeDir:             user.Conf.HomeDir,
		IsRoot:              user.Conf.IsRoot,
		Os:                  config.OSType(user.Conf.Os),
		SudoExecUser:        user.Conf.User,
		ShellTypeToLocation: user.Conf.ShellTypeToLocation,
	}
	dmn := daemon.NewDaemon(daemonConf, logging.Log)

	fmt.Fprintln(config.SysConfig.Out, "Stopping ODA daemon...")
	if err := dmn.StopDaemon(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to stop daemon")
		return errors.Wrap(err, "failed to stop ODA daemon")
	}
	fmt.Fprintln(config.SysConfig.Out, "ODA daemon stopped.")
	return nil
}

func install(cmd *cobra.Command, _ []string) error {
	setupConfig()

	// validate the stuff
	if len(installFlags.shells) > 0 {
		for _, shellType := range installFlags.shells {
			if strings.EqualFold(shellType, "all") {
				installFlags.shells = config.SupportedShells
				break
			} else if config.GetShellType(shellType) == -1 {
				fmt.Fprintf(config.SysConfig.ErrOut, "Unsupported shell: %s\nPlease choose one of: %+v\n", shellType, config.SupportedShells)
				os.Exit(1)
			}
		}
	}

	// if shells are provided, lets deal with it
	if len(installFlags.shells) > 0 {
		user.Conf.ShellTypeToLocation = make(map[config.ShellType]string)
		for _, shell := range installFlags.shells {
			user.Conf.ShellTypeToLocation[config.GetShellType(shell)] = shell
		}
	}

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
	user.ConfigureUserSystemInfo(user.Conf)

	daemonConf := &daemon.Config{
		ExePath:             user.Conf.ExePath,
		HomeDir:             user.Conf.HomeDir,
		IsRoot:              user.Conf.IsRoot,
		Os:                  config.OSType(user.Conf.Os),
		SudoExecUser:        user.Conf.User,
		AutoCredential:      autoCredentials,
		IsWorkspace:         isWorkspace,
		ShellTypeToLocation: user.Conf.ShellTypeToLocation,
		BaseCommandPath:     cmd.CommandPath(),
	}
	dmn := daemon.NewDaemon(daemonConf, logging.Log)

	fmt.Fprintln(config.SysConfig.Out, "Installing ODA daemon...")
	if err := dmn.InstallDaemonConfiguration(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to install daemon configuration")
		return errors.Wrap(err, "failed to install ODA daemon configuration file")
	}

	for shellType, shellLocation := range user.Conf.ShellTypeToLocation {
		logging.Log.Debug().Msgf("Installing shell configuration for %s", shellLocation)
		shellConfig := &shell.Config{
			ShellType:     config.ShellType(shellType),
			ShellLocation: shellLocation,
			IsRoot:        user.Conf.IsRoot,
			SudoExecUser:  user.Conf.User,
			OdaDir:        user.Conf.OdaDir,
			HomeDir:       user.Conf.HomeDir,
		}

		shl, err := shell.NewShell(shellConfig, logging.Log)

		if err != nil {
			logging.Log.Error().Err(err).Msg("Failed to setup shell")
			os.Exit(1)
		}

		if err := shl.InstallShellConfiguration(); err != nil {
			logging.Log.Error().Err(err).Msg("Failed to install shell configuration")
			return errors.Wrap(err, "failed to install ODA shell configuration files")
		}

		if err := shl.InjectShellSource(installFlags.nonInteractive); err != nil {
			logging.Log.Error().Err(err).Msgf("Failed to inject shell source (%s); will reattempt at `start` time", shellLocation)
		}
	}

	fmt.Fprintln(config.SysConfig.Out, "ODA daemon installed successfully.")
	return nil
}

func uninstall(_ *cobra.Command, _ []string) error {

	setupConfig()

	user.ConfigureUserSystemInfo(user.Conf)

	daemonConf := &daemon.Config{
		ExePath:             user.Conf.ExePath,
		HomeDir:             user.Conf.HomeDir,
		IsRoot:              user.Conf.IsRoot,
		Os:                  config.OSType(user.Conf.Os),
		SudoExecUser:        user.Conf.User,
		ShellTypeToLocation: user.Conf.ShellTypeToLocation,
	}
	dmn := daemon.NewDaemon(daemonConf, logging.Log)

	for shellType, shellLocation := range user.Conf.ShellTypeToLocation {
		shellConfig := &shell.Config{
			ShellType:     config.ShellType(shellType),
			ShellLocation: shellLocation,
			IsRoot:        user.Conf.IsRoot,
			SudoExecUser:  user.Conf.User,
			OdaDir:        user.Conf.OdaDir,
			HomeDir:       user.Conf.HomeDir,
		}
		shl, err := shell.NewShell(shellConfig, logging.Log)

		if err != nil {
			logging.Log.Error().Err(err).Msg("Failed to setup shell")
			os.Exit(1)
		}

		if err := shl.DeleteShellConfiguration(); err != nil {
			logging.Log.Error().Err(err).Msg("Failed to delete shell configuration")
			return errors.Wrap(err, "failed to delete ODA shell configuration files")
		}
	}

	fmt.Fprintln(config.SysConfig.Out, "Uninstalling ODA daemon...")
	if err := dmn.DestroyDaemonConfiguration(); err != nil {
		logging.Log.Error().Err(err).Msg("Failed to uninstall daemon configuration")
		return errors.Wrap(err, "failed to uninstall ODA daemon configuration file")
	}

	fmt.Fprintln(config.SysConfig.Out, `Daemon service files and shell configuration deleted successfully, 
		~/.oda directory still holds database file, and your rc file stills has source script.
		If you wish to remove those, delete them manually`)

	return nil
}

func displayConfig(_ *cobra.Command, _ []string) error {
	conf, err := user.GetConfig()
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to get os config")
		return errors.Wrap(err, "failed to get os config, please run 'oda install' first")
	}

	fmt.Fprintf(config.SysConfig.Out, "Current configuration: %+v \n", conf)

	return nil
}

func serve(cmd *cobra.Command, _ []string) error {
	setupConfig()

	portFlag := cmd.Flag("port").Value

	fmt.Fprintf(config.SysConfig.Out, "Serving local frontend client on http://localhost:%v\n", portFlag)

	resources.Serve()

	err := http.ListenAndServe(fmt.Sprintf(":%v", portFlag), nil)
	if err != nil {
		return errors.Wrap(err, "pass a port when calling serve; example: `oda serve -p 8987`")
	}
	return nil
}
