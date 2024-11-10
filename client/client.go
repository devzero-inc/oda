package client

import (
	"context"
	"fmt"
	"time"

	gen "github.com/devzero-inc/oda/gen/api/v1"
	"github.com/devzero-inc/oda/logging"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
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
	conn    *grpc.ClientConn
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
	var opts []grpc.DialOption

	// Setup connection security based on config
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if c.config.SecureConnection {
		if c.config.CertFile != "" {
			tlsFromFile, err := credentials.NewClientTLSFromFile(c.config.CertFile, "")
			if err != nil {
				return fmt.Errorf("failed to create TLS credentials: %w", err)
			}
			creds = grpc.WithTransportCredentials(tlsFromFile)
		} else {
			creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
		}
	}
	opts = append(opts, creds)

	// Adding keepalive parameters to manage connection health
	keepAliveParams := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second, // Ping the server every 10 seconds to keep the connection alive
		Timeout:             5 * time.Second,  // Wait 5 seconds for a pong before closing the connection
		PermitWithoutStream: true,             // Send pings even without active RPCs
	})
	opts = append(opts, keepAliveParams)

	// Dial the server
	conn, err := grpc.Dial(c.config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Set the connection on the client
	c.conn = conn
	return nil
}

// Reconnect attempts to reconnect if the connection is down
func (c *Client) Reconnect() error {
	if c.conn != nil {
		c.Close()
	}
	return c.connect()
}

// CheckAndReconnect checks connection health and reconnects if necessary
func (c *Client) CheckAndReconnect() error {
	if c.conn.GetState() == connectivity.TransientFailure || c.conn.GetState() == connectivity.Shutdown {
		c.logger.Warn().Msg("Connection lost. Attempting to reconnect...")
		return c.Reconnect()
	}
	return nil
}

// SendCommands sends a list of commands to the server
func (c *Client) SendCommands(commands []*gen.Command, auth *gen.Auth) error {

	if err := c.CheckAndReconnect(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	client := gen.NewCollectorServiceClient(c.conn)

	req := &gen.SendCommandsRequest{
		Commands: commands,
		Auth:     auth,
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := client.SendCommands(ctx, req)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to send commands")
	}

	return err
}

// SendProcesses sends a list of processes to the server
func (c *Client) SendProcesses(processes []*gen.Process, auth *gen.Auth) error {

	if err := c.CheckAndReconnect(); err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	client := gen.NewCollectorServiceClient(c.conn)

	req := &gen.SendProcessesRequest{
		Processes: processes,
		Auth:      auth,
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	_, err := client.SendProcesses(ctx, req)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to send processes")
	}

	return err
}

// Close closes the connection to the server
func (c *Client) Close() {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			c.logger.Error().Err(err).Msg("Failed to close the connection")
		} else {
			c.logger.Info().Msg("Connection closed successfully")
		}
	}
}
