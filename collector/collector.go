package collector

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/devzero-inc/oda/client"
	gen "github.com/devzero-inc/oda/gen/api/v1"
	"github.com/devzero-inc/oda/process"
	"github.com/devzero-inc/oda/util"

	"github.com/rs/zerolog"

	"time"
)

// TODO move this to /var/run or other appropriate location based on OS,
// TODO /var/run has issues with permisisons os have to explore a bit more.
const SocketPath = "/tmp/oda.socket"

// Collector collects command and system information
type Collector struct {
	socketPath       string
	client           *client.Client
	logger           zerolog.Logger
	excludeRegex     string
	collectionConfig collectionConfig
	authConfig       AuthConfig
	protoAuthConfig  *gen.Auth
	intervalConfig   IntervalConfig
}

// IntervalConfig contains the configuration for the collection intervals
type IntervalConfig struct {
	ProcessInterval           time.Duration
	CommandInterval           time.Duration
	CommandIntervalMultiplier float64
	MaxConcurrentCommands     int
	MaxDuration               time.Duration
}

// AuthConfig contains the configuration for the command processing and authentication
type AuthConfig struct {
	TeamID      string
	UserID      string
	WorkspaceID string
	UserEmail   string
}

// collectionConfig contains the configuration for the collection process
type collectionConfig struct {
	// ongoingCommands is a map of currently running commands
	ongoingCommands map[string]Command
	// collectionMutex is a mutex to protect the ongoingCommands map
	collectionMutex sync.Mutex
	// activeCommandsCounter is a counter for the number of active commands
	activeCommandsCounter int
	// collectionContext is the context for the collection process
	collectionContext context.Context
	// collectionCancelFunc is the cancel function for the collection context
	collectionCancelFunc context.CancelFunc
	// isCollectionRunning is a flag to indicate if the collection is running
	isCollectionRunning bool
	// process is the system process collector
	process process.SystemProcess
}

// NewCollector creates a new collector instance
func NewCollector(socketPath string, client *client.Client, logger zerolog.Logger, config IntervalConfig, auth AuthConfig, excludeRegex string, process process.SystemProcess) *Collector {

	collector := &Collector{
		socketPath: socketPath,
		client:     client,
		logger:     logger,
		collectionConfig: collectionConfig{
			ongoingCommands: make(map[string]Command),
			process:         process,
		},
		intervalConfig: config,
		authConfig:     auth,
		excludeRegex:   excludeRegex,
	}

	if auth.TeamID != "" && auth.UserEmail != "" {
		collector.protoAuthConfig = &gen.Auth{
			UserId:      auth.UserID,
			TeamId:      auth.TeamID,
			WorkspaceId: &auth.WorkspaceID,
			UserEmail:   auth.UserEmail,
		}
	}

	return collector
}

// Collect starts the collection of command and system information
func (c *Collector) Collect() {
	c.logger.Info().Msg("Collecting command and system information")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.collectSystemInformation(ctx, c.intervalConfig.ProcessInterval, 3, c.intervalConfig.MaxDuration)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.collectCommandInformation(); err != nil {
			c.logger.Error().Err(err).Msg("Failed to collect command information")
			cancel()
		}
	}()

	wg.Wait()

	c.logger.Info().Msg("Collection stopped")
}

// collectSystemInformation uses exponential backoff for intervals between collections.
func (c *Collector) collectSystemInformation(ctx context.Context, initialDuration time.Duration, increaseFactor float64, maxDuration time.Duration) {
	currentDuration := initialDuration

	for {
		select {
		case <-ctx.Done():
			c.logger.Debug().Msg("Shutting down collection of system information")
			return
		case <-time.After(currentDuration):
			// Perform the collection on each tick
			if err := c.collectOnce(); err != nil {
				c.logger.Error().Err(err).Msg("Failed to collect system information")
			}

			// Calculate the next interval with exponential backoff
			currentDuration = time.Duration(float64(currentDuration) * increaseFactor)
			if currentDuration > maxDuration {
				currentDuration = maxDuration
			}

			c.logger.Debug().Msgf("Next collection in %s", currentDuration)
		}
	}
}

func (c *Collector) collectOnce() error {

	c.logger.Debug().Msg("Collecting process")

	processes, err := c.collectionConfig.process.Collect()
	if err != nil {
		c.logger.Err(err).Msg("Failed to collect processes")
		return err
	}

	if err := process.InsertProcesses(processes); err != nil {
		c.logger.Error().Err(err).Msg("Failed to insert processes")
	}

	if c.client != nil {
		var processMetrics []*gen.Process
		for _, p := range processes {

			processMetrics = append(
				processMetrics,
				process.MapProcessToProto(p),
			)
		}

		go func() {
			if err := c.client.SendProcesses(processMetrics, c.protoAuthConfig); err != nil {
				c.logger.Error().Err(err).Msg("Failed to send processes")
			}
		}()
	}

	return nil
}

func (c *Collector) onStartCommand() {
	c.collectionConfig.collectionMutex.Lock()
	defer c.collectionConfig.collectionMutex.Unlock()

	// Perform initial collection for every command
	if err := c.collectOnce(); err != nil {
		c.logger.Error().Err(err).Msg("Failed to collect system information")
	}

	c.collectionConfig.activeCommandsCounter++
	// If the collection is not running, start it with a timeout
	if !c.collectionConfig.isCollectionRunning {
		c.logger.Debug().Msg("Starting collection")
		c.collectionConfig.collectionContext, c.collectionConfig.collectionCancelFunc =
			context.WithTimeout(context.Background(), c.intervalConfig.MaxDuration)
		go c.collectSystemInformation(
			c.collectionConfig.collectionContext,
			c.intervalConfig.CommandInterval,
			c.intervalConfig.CommandIntervalMultiplier,
			c.intervalConfig.MaxDuration,
		)
		c.collectionConfig.isCollectionRunning = true
	}
}

func (c *Collector) onEndCommand() {
	c.collectionConfig.collectionMutex.Lock()
	defer c.collectionConfig.collectionMutex.Unlock()

	c.collectionConfig.activeCommandsCounter--
	// If there are no more active commands, stop the collection
	if c.collectionConfig.activeCommandsCounter == 0 && c.collectionConfig.isCollectionRunning {
		c.logger.Debug().Msg("Stopping collection")
		c.collectionConfig.collectionCancelFunc()
		c.collectionConfig.isCollectionRunning = false
	}
}

func (c *Collector) collectCommandInformation() error {
	if err := util.Fs.RemoveAll(SocketPath); err != nil {
		c.logger.Error().Err(err).Msg("Failed to clean up existing socket")
		return err
	}

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to listen on UNIX socket")
		return err
	}
	defer listener.Close()

	// Limit the number of concurrent goroutines handling connections
	semaphore := make(chan struct{}, c.intervalConfig.MaxConcurrentCommands)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-ctx.Done() // Wait for context cancellation
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// If the context is canceled, stop accepting new connections
				return nil
			default:
				c.logger.Error().Err(err).Msg("Failed to accept connection")
				continue
			}
		}

		semaphore <- struct{}{} // Acquire
		go func(conn net.Conn) {
			defer func() {
				<-semaphore // Release
			}()
			if err := c.handleSocketCollection(conn); err != nil {
				c.logger.Error().Err(err).Msg("Error handling socket collection")
			}
		}(conn)
	}
}

func (c *Collector) handleSocketCollection(con net.Conn) error {
	defer con.Close()
	var buf [1024]byte
	n, err := con.Read(buf[:])
	if err != nil {
		c.logger.Error().Err(err).Msg("Error reading from socket")
		return err
	}

	data := string(buf[:n])
	parts := strings.Split(data, "|")

	c.logger.Debug().Msgf("Received: %s", string(buf[:n]))

	if len(parts) != 8 {
		c.logger.Error().Msg("Invalid command format")
		return fmt.Errorf("invalid command format")
	}

	if parts[0] == "start" {
		if err := c.handleStartCommand(parts); err != nil {
			c.logger.Error().Err(err).Msg("Error handling start command")
		}
	} else if parts[0] == "end" {
		if err := c.handleEndCommand(parts); err != nil {
			c.logger.Error().Err(err).Msg("Error handling end command")
		}
	} else {
		c.logger.Error().Msg("Invalid command format")
		return err
	}

	return nil
}

func (c *Collector) handleStartCommand(parts []string) error {
	if !IsCommandAcceptable(parts[1], c.excludeRegex) {
		c.logger.Debug().Msg("Command is not acceptable")
		return fmt.Errorf("command is not acceptable")
	}

	c.logger.Debug().Msgf("Parsing command: %s", parts[0])

	repo, err := util.GetRepoNameFromConfig(parts[2])
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get repository name")
	}

	pid, _ := strconv.ParseInt(parts[5], 10, 64)

	command := Command{
		Category:   ParseCommand(parts[1]),
		Command:    parts[1],
		Directory:  parts[2],
		User:       parts[3],
		StartTime:  time.Now().UnixMilli(), // TODO: there are some issues with sending time through shell because of ms support on MAC, explore more
		Repository: repo,
		PID:        pid,
	}

	c.collectionConfig.ongoingCommands[parts[4]] = command

	c.onStartCommand()

	return nil
}

func (c *Collector) handleEndCommand(parts []string) error {

	if !IsCommandAcceptable(parts[1], c.excludeRegex) {
		c.logger.Debug().Msg("Command is not acceptable")
		return fmt.Errorf("command is not acceptable")
	}

	c.logger.Debug().Msgf("Parsing command: %s", parts[0])

	if command, exists := c.collectionConfig.ongoingCommands[parts[4]]; exists {
		command.EndTime = time.Now().UnixMilli()
		command.ExecutionTime = command.EndTime - command.StartTime
		command.Result = parts[6]
		command.Status = parts[7]

		c.logger.Debug().Msgf("Command: %+v", command)
		if err := InsertCommand(command); err != nil {
			c.logger.Error().Err(err).Msg("Failed to insert command")
			return err
		}

		delete(c.collectionConfig.ongoingCommands, parts[4])
		c.onEndCommand()

		if c.client != nil {
			go func() {
				if err := c.client.SendCommands([]*gen.Command{MapCommandToProto(command)}, c.protoAuthConfig); err != nil {
					c.logger.Error().Err(err).Msg("Failed to send command")
				}
			}()
		}
	} else {
		c.logger.Error().Msg("Matching start command not found")
		return fmt.Errorf("matching start command not found")
	}

	return nil
}
