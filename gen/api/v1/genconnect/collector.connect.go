// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: api/v1/collector.proto

package genconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/devzero-inc/oda/gen/api/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion0_1_0

const (
	// CollectorServiceName is the fully-qualified name of the CollectorService service.
	CollectorServiceName = "api.v1.CollectorService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// CollectorServiceSendCommandsProcedure is the fully-qualified name of the CollectorService's
	// SendCommands RPC.
	CollectorServiceSendCommandsProcedure = "/api.v1.CollectorService/SendCommands"
	// CollectorServiceSendProcessesProcedure is the fully-qualified name of the CollectorService's
	// SendProcesses RPC.
	CollectorServiceSendProcessesProcedure = "/api.v1.CollectorService/SendProcesses"
)

// CollectorServiceClient is a client for the api.v1.CollectorService service.
type CollectorServiceClient interface {
	// RPC method for sending command data.
	SendCommands(context.Context, *connect.Request[v1.SendCommandsRequest]) (*connect.Response[emptypb.Empty], error)
	// RPC method for sending process data.
	SendProcesses(context.Context, *connect.Request[v1.SendProcessesRequest]) (*connect.Response[emptypb.Empty], error)
}

// NewCollectorServiceClient constructs a client for the api.v1.CollectorService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewCollectorServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) CollectorServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &collectorServiceClient{
		sendCommands: connect.NewClient[v1.SendCommandsRequest, emptypb.Empty](
			httpClient,
			baseURL+CollectorServiceSendCommandsProcedure,
			opts...,
		),
		sendProcesses: connect.NewClient[v1.SendProcessesRequest, emptypb.Empty](
			httpClient,
			baseURL+CollectorServiceSendProcessesProcedure,
			opts...,
		),
	}
}

// collectorServiceClient implements CollectorServiceClient.
type collectorServiceClient struct {
	sendCommands  *connect.Client[v1.SendCommandsRequest, emptypb.Empty]
	sendProcesses *connect.Client[v1.SendProcessesRequest, emptypb.Empty]
}

// SendCommands calls api.v1.CollectorService.SendCommands.
func (c *collectorServiceClient) SendCommands(ctx context.Context, req *connect.Request[v1.SendCommandsRequest]) (*connect.Response[emptypb.Empty], error) {
	return c.sendCommands.CallUnary(ctx, req)
}

// SendProcesses calls api.v1.CollectorService.SendProcesses.
func (c *collectorServiceClient) SendProcesses(ctx context.Context, req *connect.Request[v1.SendProcessesRequest]) (*connect.Response[emptypb.Empty], error) {
	return c.sendProcesses.CallUnary(ctx, req)
}

// CollectorServiceHandler is an implementation of the api.v1.CollectorService service.
type CollectorServiceHandler interface {
	// RPC method for sending command data.
	SendCommands(context.Context, *connect.Request[v1.SendCommandsRequest]) (*connect.Response[emptypb.Empty], error)
	// RPC method for sending process data.
	SendProcesses(context.Context, *connect.Request[v1.SendProcessesRequest]) (*connect.Response[emptypb.Empty], error)
}

// NewCollectorServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewCollectorServiceHandler(svc CollectorServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	collectorServiceSendCommandsHandler := connect.NewUnaryHandler(
		CollectorServiceSendCommandsProcedure,
		svc.SendCommands,
		opts...,
	)
	collectorServiceSendProcessesHandler := connect.NewUnaryHandler(
		CollectorServiceSendProcessesProcedure,
		svc.SendProcesses,
		opts...,
	)
	return "/api.v1.CollectorService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case CollectorServiceSendCommandsProcedure:
			collectorServiceSendCommandsHandler.ServeHTTP(w, r)
		case CollectorServiceSendProcessesProcedure:
			collectorServiceSendProcessesHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedCollectorServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedCollectorServiceHandler struct{}

func (UnimplementedCollectorServiceHandler) SendCommands(context.Context, *connect.Request[v1.SendCommandsRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("api.v1.CollectorService.SendCommands is not implemented"))
}

func (UnimplementedCollectorServiceHandler) SendProcesses(context.Context, *connect.Request[v1.SendProcessesRequest]) (*connect.Response[emptypb.Empty], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("api.v1.CollectorService.SendProcesses is not implemented"))
}
