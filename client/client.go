package client

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	gen "github.com/devzero-inc/oda/gen/api/v1"
	"github.com/devzero-inc/oda/logging"

	"github.com/rs/zerolog"

	genConnect "github.com/devzero-inc/oda/gen/api/v1/genconnect"
)

// Config holds configuration for the client connection.
type Config struct {
	Address          string // The server address
	SecureConnection bool   // True for secure (HTTPS), false for insecure (HTTP)
	CertFile         string // Optional path to the TLS cert file for secure connections
	Timeout          int    // Timeout in seconds for the connection
}

// Client is a struct that holds the connection to the server
type Client struct {
	client  genConnect.CollectorServiceClient
	logger  *zerolog.Logger
	timeout time.Duration
	config  Config
}

// NewClient creates a new client with connection management and returns a pointer to it and an error
func NewClient(config Config) (*Client, error) {
	client := &Client{
		logger:  &logging.Log,
		timeout: time.Duration(config.Timeout) * time.Second,
		config:  config,
	}

	// Establish the initial connection
	err := client.connect()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// connect handles connection establishment and configuration
func (c *Client) connect() error {
	c.client = genConnect.NewCollectorServiceClient(http.DefaultClient, c.config.Address, connect.WithGRPC())

	return nil
}

// SendCommands sends a list of commands to the server
func (c *Client) SendCommands(commands []*gen.Command, auth *gen.Auth) error {

	c.logger.Debug().Msg("Sending commands")

	req := &gen.SendCommandsRequest{
		Commands: commands,
		Auth:     auth,
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.SendCommands(ctx, connect.NewRequest(req))
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to send commands")
	}

	return err
}

// SendProcesses sends a list of processes to the server
func (c *Client) SendProcesses(processes []*gen.Process, auth *gen.Auth) error {

	c.logger.Debug().Msg("Sending processes")

	req := &gen.SendProcessesRequest{
		Processes: processes,
		Auth:      auth,
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := c.client.SendProcesses(ctx, connect.NewRequest(req))
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to send processes")
	}

	return err
}
