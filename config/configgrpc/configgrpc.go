// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configgrpc // import "go.opentelemetry.io/collector/config/configgrpc"

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mostynb/go-grpc-compression/nonclobbering/snappy"
	"github.com/mostynb/go-grpc-compression/nonclobbering/zstd"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configmiddleware"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

var errMetadataNotFound = errors.New("no request metadata found")

// KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter:
// https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters
type KeepaliveClientConfig struct {
	Time                time.Duration `mapstructure:"time"`
	Timeout             time.Duration `mapstructure:"timeout"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultKeepaliveClientConfig returns a new instance of KeepaliveClientConfig with default values.
func NewDefaultKeepaliveClientConfig() KeepaliveClientConfig {
	return KeepaliveClientConfig{
		Time:    time.Second * 10,
		Timeout: time.Second * 10,
	}
}

// BalancerName returns a string with default load balancer value
func BalancerName() string {
	return "round_robin"
}

// ClientConfig defines common settings for a gRPC client configuration.
type ClientConfig struct {
	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `mapstructure:"endpoint,omitempty"`

	// The compression key for supported compression types within collector.
	Compression configcompression.Type `mapstructure:"compression,omitempty"`

	// TLS struct exposes TLS client configuration.
	TLS configtls.ClientConfig `mapstructure:"tls,omitempty"`

	// The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive configoptional.Optional[KeepaliveClientConfig] `mapstructure:"keepalive,omitempty"`

	// ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
	ReadBufferSize int `mapstructure:"read_buffer_size,omitempty"`

	// WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
	WriteBufferSize int `mapstructure:"write_buffer_size,omitempty"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `mapstructure:"wait_for_ready,omitempty"`

	// The headers associated with gRPC requests.
	Headers configopaque.MapList `mapstructure:"headers,omitempty"`

	// Sets the balancer in grpclb_policy to discover the servers. Default is pick_first.
	// https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
	BalancerName string `mapstructure:"balancer_name"`

	// WithAuthority parameter configures client to rewrite ":authority" header
	// (godoc.org/google.golang.org/grpc#WithAuthority)
	Authority string `mapstructure:"authority,omitempty"`

	// Auth configuration for outgoing RPCs.
	Auth configoptional.Optional[configauth.Config] `mapstructure:"auth,omitempty"`

	// Middlewares for the gRPC client.
	Middlewares []configmiddleware.Config `mapstructure:"middlewares,omitempty"`
}

// NewDefaultClientConfig returns a new instance of ClientConfig with default values.
func NewDefaultClientConfig() ClientConfig {
	return ClientConfig{
		TLS:          configtls.NewDefaultClientConfig(),
		Keepalive:    configoptional.Some(NewDefaultKeepaliveClientConfig()),
		BalancerName: BalancerName(),
	}
}

// KeepaliveServerConfig is the configuration for keepalive.
type KeepaliveServerConfig struct {
	ServerParameters  configoptional.Optional[KeepaliveServerParameters]  `mapstructure:"server_parameters,omitempty"`
	EnforcementPolicy configoptional.Optional[KeepaliveEnforcementPolicy] `mapstructure:"enforcement_policy,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultKeepaliveServerConfig returns a new instance of KeepaliveServerConfig with default values.
func NewDefaultKeepaliveServerConfig() KeepaliveServerConfig {
	return KeepaliveServerConfig{
		ServerParameters:  configoptional.Some(NewDefaultKeepaliveServerParameters()),
		EnforcementPolicy: configoptional.Some(NewDefaultKeepaliveEnforcementPolicy()),
	}
}

// KeepaliveServerParameters allow configuration of the keepalive.ServerParameters.
// The same default values as keepalive.ServerParameters are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters for details.
type KeepaliveServerParameters struct {
	MaxConnectionIdle     time.Duration `mapstructure:"max_connection_idle,omitempty"`
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age,omitempty"`
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace,omitempty"`
	Time                  time.Duration `mapstructure:"time,omitempty"`
	Timeout               time.Duration `mapstructure:"timeout,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultKeepaliveServerParameters creates and returns a new instance of KeepaliveServerParameters with default settings.
func NewDefaultKeepaliveServerParameters() KeepaliveServerParameters {
	return KeepaliveServerParameters{}
}

// KeepaliveEnforcementPolicy allow configuration of the keepalive.EnforcementPolicy.
// The same default values as keepalive.EnforcementPolicy are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy for details.
type KeepaliveEnforcementPolicy struct {
	MinTime             time.Duration `mapstructure:"min_time,omitempty"`
	PermitWithoutStream bool          `mapstructure:"permit_without_stream,omitempty"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultKeepaliveEnforcementPolicy creates and returns a new instance of KeepaliveEnforcementPolicy with default settings.
func NewDefaultKeepaliveEnforcementPolicy() KeepaliveEnforcementPolicy {
	return KeepaliveEnforcementPolicy{}
}

// ServerConfig defines common settings for a gRPC server configuration.
type ServerConfig struct {
	// Server net.Addr config. For transport only "tcp" and "unix" are valid options.
	NetAddr confignet.AddrConfig `mapstructure:",squash"`

	// Configures the protocol to use TLS.
	// The default value is nil, which will cause the protocol to not use TLS.
	TLS configoptional.Optional[configtls.ServerConfig] `mapstructure:"tls,omitempty"`

	// MaxRecvMsgSizeMiB sets the maximum size (in MiB) of messages accepted by the server.
	MaxRecvMsgSizeMiB int `mapstructure:"max_recv_msg_size_mib,omitempty"`

	// MaxConcurrentStreams sets the limit on the number of concurrent streams to each ServerTransport.
	// It has effect only for streaming RPCs.
	MaxConcurrentStreams uint32 `mapstructure:"max_concurrent_streams,omitempty,omitempty"`

	// ReadBufferSize for gRPC server. See grpc.ReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#ReadBufferSize).
	ReadBufferSize int `mapstructure:"read_buffer_size,omitempty"`

	// WriteBufferSize for gRPC server. See grpc.WriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WriteBufferSize).
	WriteBufferSize int `mapstructure:"write_buffer_size,omitempty"`

	// Keepalive anchor for all the settings related to keepalive.
	Keepalive configoptional.Optional[KeepaliveServerConfig] `mapstructure:"keepalive,omitempty"`

	// Auth for this receiver
	Auth configoptional.Optional[configauth.Config] `mapstructure:"auth,omitempty"`

	// Include propagates the incoming connection's metadata to downstream consumers.
	IncludeMetadata bool `mapstructure:"include_metadata,omitempty"`

	// Middlewares for the gRPC server.
	Middlewares []configmiddleware.Config `mapstructure:"middlewares,omitempty"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// NewDefaultServerConfig returns a new instance of ServerConfig with default values.
func NewDefaultServerConfig() ServerConfig {
	netAddr := confignet.NewDefaultAddrConfig()

	// We typically want to create a TCP server and listen over a network.
	netAddr.Transport = confignet.TransportTypeTCP

	return ServerConfig{
		Keepalive: configoptional.Some(NewDefaultKeepaliveServerConfig()),
		NetAddr:   netAddr,
	}
}

func (cc *ClientConfig) Validate() error {
	if after, ok := strings.CutPrefix(cc.Endpoint, "unix://"); ok {
		if after == "" {
			return errors.New("unix socket path cannot be empty")
		}
		return nil
	}

	if endpoint := cc.sanitizedEndpoint(); endpoint != "" {
		// Validate that the port is in the address
		_, port, err := net.SplitHostPort(endpoint)
		if err != nil {
			return err
		}
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf(`invalid port "%v"`, port)
		}
	}

	if cc.BalancerName != "" {
		if balancer.Get(cc.BalancerName) == nil {
			return fmt.Errorf("invalid balancer_name: %s", cc.BalancerName)
		}
	}

	return nil
}

// sanitizedEndpoint strips the prefix of either http:// or https:// from configgrpc.ClientConfig.Endpoint.
func (cc *ClientConfig) sanitizedEndpoint() string {
	switch {
	case cc.isSchemeHTTP():
		return strings.TrimPrefix(cc.Endpoint, "http://")
	case cc.isSchemeHTTPS():
		return strings.TrimPrefix(cc.Endpoint, "https://")
	case strings.HasPrefix(cc.Endpoint, "dns://"):
		r := regexp.MustCompile(`^dns:///?`)
		return r.ReplaceAllString(cc.Endpoint, "")
	default:
		return cc.Endpoint
	}
}

func (cc *ClientConfig) isSchemeHTTP() bool {
	return strings.HasPrefix(cc.Endpoint, "http://")
}

func (cc *ClientConfig) isSchemeHTTPS() bool {
	return strings.HasPrefix(cc.Endpoint, "https://")
}

// ToClientConnOption is a sealed interface wrapping options for [ClientConfig.ToClientConn].
type ToClientConnOption interface {
	isToClientConnOption()
}

type grpcDialOptionWrapper struct {
	opt grpc.DialOption
}

// WithGrpcDialOption wraps a [grpc.DialOption] into a [ToClientConnOption].
func WithGrpcDialOption(opt grpc.DialOption) ToClientConnOption {
	return grpcDialOptionWrapper{opt: opt}
}
func (grpcDialOptionWrapper) isToClientConnOption() {}

// ToClientConn creates a client connection to the given target. By default, it's
// a non-blocking dial (the function won't wait for connections to be
// established, and connecting happens in the background). To make it a blocking
// dial, use the WithGrpcDialOption(grpc.WithBlock()) option.
//
// To allow the configuration to reference middleware or authentication extensions,
// the `extensions` argument should be the output of `host.GetExtensions()`.
// It may also be `nil` in tests where no such extension is expected to be used.
func (cc *ClientConfig) ToClientConn(
	ctx context.Context,
	extensions map[component.ID]component.Component,
	settings component.TelemetrySettings,
	extraOpts ...ToClientConnOption,
) (*grpc.ClientConn, error) {
	grpcOpts, err := cc.getGrpcDialOptions(ctx, extensions, settings, extraOpts)
	if err != nil {
		return nil, err
	}
	//nolint:staticcheck // SA1019 see https://github.com/open-telemetry/opentelemetry-collector/pull/11575
	return grpc.DialContext(ctx, cc.sanitizedEndpoint(), grpcOpts...)
}

func (cc *ClientConfig) addHeadersIfAbsent(ctx context.Context) context.Context {
	kv := make([]string, 0, 2*len(cc.Headers))
	existingMd, _ := metadata.FromOutgoingContext(ctx)
	for k, v := range cc.Headers.Iter {
		if len(existingMd.Get(k)) == 0 {
			kv = append(kv, k, string(v))
		}
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}

func (cc *ClientConfig) getGrpcDialOptions(
	ctx context.Context,
	extensions map[component.ID]component.Component,
	settings component.TelemetrySettings,
	extraOpts []ToClientConnOption,
) ([]grpc.DialOption, error) {
	var opts []grpc.DialOption
	if cc.Compression.IsCompressed() {
		cp, err := getGRPCCompressionName(cc.Compression)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(cp)))
	}

	tlsCfg, err := cc.TLS.LoadTLSConfig(ctx)
	if err != nil {
		return nil, err
	}
	cred := insecure.NewCredentials()
	if tlsCfg != nil {
		cred = credentials.NewTLS(tlsCfg)
	} else if cc.isSchemeHTTPS() {
		cred = credentials.NewTLS(&tls.Config{})
	}
	opts = append(opts, grpc.WithTransportCredentials(cred))

	if cc.ReadBufferSize > 0 {
		opts = append(opts, grpc.WithReadBufferSize(cc.ReadBufferSize))
	}

	if cc.WriteBufferSize > 0 {
		opts = append(opts, grpc.WithWriteBufferSize(cc.WriteBufferSize))
	}

	if cc.Keepalive.HasValue() {
		keepaliveConfig := cc.Keepalive.Get()
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepaliveConfig.Time,
			Timeout:             keepaliveConfig.Timeout,
			PermitWithoutStream: keepaliveConfig.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}

	if cc.Auth.HasValue() {
		if extensions == nil {
			return nil, errors.New("authentication was configured but this component or its host does not support extensions")
		}

		grpcAuthenticator, cerr := cc.Auth.Get().GetGRPCClientAuthenticator(ctx, extensions)
		if cerr != nil {
			return nil, cerr
		}

		perRPCCredentials, perr := grpcAuthenticator.PerRPCCredentials()
		if perr != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithPerRPCCredentials(perRPCCredentials))
	}

	if cc.BalancerName != "" {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":%q}`, cc.BalancerName)))
	}

	if cc.Authority != "" {
		opts = append(opts, grpc.WithAuthority(cc.Authority))
	}

	otelOpts := []otelgrpc.Option{
		otelgrpc.WithTracerProvider(settings.TracerProvider),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		otelgrpc.WithMeterProvider(settings.MeterProvider),
	}

	// Enable OpenTelemetry observability plugin.
	opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler(otelOpts...)))

	if len(cc.Headers) > 0 {
		opts = append(opts,
			grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, gcc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return invoker(cc.addHeadersIfAbsent(ctx), method, req, reply, gcc, opts...)
			}),
			grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, gcc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				return streamer(cc.addHeadersIfAbsent(ctx), desc, gcc, method, opts...)
			}),
		)
	}

	// Apply middleware options. Note: OpenTelemetry could be registered as an extension.
	if len(cc.Middlewares) > 0 && extensions == nil {
		return nil, errors.New("middlewares were configured but this component or its host does not support extensions")
	}
	for _, middleware := range cc.Middlewares {
		middlewareOptions, err := middleware.GetGRPCClientOptions(ctx, extensions)
		if err != nil {
			return nil, fmt.Errorf("failed to get gRPC client options from middleware: %w", err)
		}
		opts = append(opts, middlewareOptions...)
	}

	for _, opt := range extraOpts {
		if wrapper, ok := opt.(grpcDialOptionWrapper); ok {
			opts = append(opts, wrapper.opt)
		}
	}

	return opts, nil
}

func (sc *ServerConfig) Validate() error {
	if sc.MaxRecvMsgSizeMiB*1024*1024 < 0 {
		return fmt.Errorf("invalid max_recv_msg_size_mib value, must be between 1 and %d: %d", math.MaxInt/1024/1024, sc.MaxRecvMsgSizeMiB)
	}

	if sc.ReadBufferSize < 0 {
		return fmt.Errorf("invalid read_buffer_size value: %d", sc.ReadBufferSize)
	}

	if sc.WriteBufferSize < 0 {
		return fmt.Errorf("invalid write_buffer_size value: %d", sc.WriteBufferSize)
	}

	return nil
}

// ToServerOption is a sealed interface wrapping options for [ServerConfig.ToServer].
type ToServerOption interface {
	isToServerOption()
}

type grpcServerOptionWrapper struct {
	opt grpc.ServerOption
}

// WithGrpcServerOption wraps a [grpc.ServerOption] into a [ToServerOption].
func WithGrpcServerOption(opt grpc.ServerOption) ToServerOption {
	return grpcServerOptionWrapper{opt: opt}
}
func (grpcServerOptionWrapper) isToServerOption() {}

// ToServer returns a [grpc.Server] for the configuration.
//
// To allow the configuration to reference middleware or authentication extensions,
// the `extensions` argument should be the output of `host.GetExtensions()`.
// It may also be `nil` in tests where no such extension is expected to be used.
func (sc *ServerConfig) ToServer(
	ctx context.Context,
	extensions map[component.ID]component.Component,
	settings component.TelemetrySettings,
	extraOpts ...ToServerOption,
) (*grpc.Server, error) {
	grpcOpts, err := sc.getGrpcServerOptions(ctx, extensions, settings, extraOpts)
	if err != nil {
		return nil, err
	}
	return grpc.NewServer(grpcOpts...), nil
}

func (sc *ServerConfig) getGrpcServerOptions(
	ctx context.Context,
	extensions map[component.ID]component.Component,
	settings component.TelemetrySettings,
	extraOpts []ToServerOption,
) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if sc.TLS.HasValue() {
		tlsCfg, err := sc.TLS.Get().LoadTLSConfig(ctx)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	if sc.MaxRecvMsgSizeMiB > 0 && sc.MaxRecvMsgSizeMiB*1024*1024 > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(sc.MaxRecvMsgSizeMiB*1024*1024))
	}

	if sc.MaxConcurrentStreams > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(sc.MaxConcurrentStreams))
	}

	if sc.ReadBufferSize > 0 {
		opts = append(opts, grpc.ReadBufferSize(sc.ReadBufferSize))
	}

	if sc.WriteBufferSize > 0 {
		opts = append(opts, grpc.WriteBufferSize(sc.WriteBufferSize))
	}

	// The default values referenced in the GRPC docs are set within the server, so this code doesn't need
	// to apply them over zero/nil values before passing these as grpc.ServerOptions.
	// The following shows the server code for applying default grpc.ServerOptions.
	// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L184-L200
	if sc.Keepalive.HasValue() {
		keepaliveConfig := sc.Keepalive.Get()
		if keepaliveConfig.ServerParameters.HasValue() {
			svrParams := keepaliveConfig.ServerParameters.Get()
			opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     svrParams.MaxConnectionIdle,
				MaxConnectionAge:      svrParams.MaxConnectionAge,
				MaxConnectionAgeGrace: svrParams.MaxConnectionAgeGrace,
				Time:                  svrParams.Time,
				Timeout:               svrParams.Timeout,
			}))
		}
		// The default values referenced in the GRPC are set within the server, so this code doesn't need
		// to apply them over zero/nil values before passing these as grpc.ServerOptions.
		// The following shows the server code for applying default grpc.ServerOptions.
		// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L202-L205
		if keepaliveConfig.EnforcementPolicy.HasValue() {
			enfPol := keepaliveConfig.EnforcementPolicy.Get()
			opts = append(opts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             enfPol.MinTime,
				PermitWithoutStream: enfPol.PermitWithoutStream,
			}))
		}
	}

	var uInterceptors []grpc.UnaryServerInterceptor
	var sInterceptors []grpc.StreamServerInterceptor

	if sc.Auth.HasValue() {
		authenticator, err := sc.Auth.Get().GetServerAuthenticator(ctx, extensions)
		if err != nil {
			return nil, err
		}

		uInterceptors = append(uInterceptors, authUnaryServerInterceptor(authenticator))
		sInterceptors = append(sInterceptors, authStreamServerInterceptor(authenticator)) //nolint:contextcheck // context already handled
	}

	otelOpts := []otelgrpc.Option{
		otelgrpc.WithTracerProvider(settings.TracerProvider),
		otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		otelgrpc.WithMeterProvider(settings.MeterProvider),
	}

	// Enable OpenTelemetry observability plugin.

	uInterceptors = append(uInterceptors, enhanceWithClientInformation(sc.IncludeMetadata))
	sInterceptors = append(sInterceptors, enhanceStreamWithClientInformation(sc.IncludeMetadata)) //nolint:contextcheck // context already handled

	opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler(otelOpts...)), grpc.ChainUnaryInterceptor(uInterceptors...), grpc.ChainStreamInterceptor(sInterceptors...))

	// Apply middleware options. Note: OpenTelemetry could be registered as an extension.
	for _, middleware := range sc.Middlewares {
		middlewareOptions, err := middleware.GetGRPCServerOptions(ctx, extensions)
		if err != nil {
			return nil, fmt.Errorf("failed to get gRPC server options from middleware: %w", err)
		}
		opts = append(opts, middlewareOptions...)
	}

	for _, opt := range extraOpts {
		if wrapper, ok := opt.(grpcServerOptionWrapper); ok {
			opts = append(opts, wrapper.opt)
		}
	}

	return opts, nil
}

// getGRPCCompressionName returns compression name registered in grpc.
func getGRPCCompressionName(compressionType configcompression.Type) (string, error) {
	switch compressionType {
	case configcompression.TypeGzip:
		return gzip.Name, nil
	case configcompression.TypeSnappy:
		return snappy.Name, nil
	case configcompression.TypeZstd:
		return zstd.Name, nil
	default:
		return "", fmt.Errorf("unsupported compression type %q", compressionType)
	}
}

// enhanceWithClientInformation intercepts the incoming RPC, replacing the incoming context with one that includes
// a client.Info, potentially with the peer's address.
func enhanceWithClientInformation(includeMetadata bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(contextWithClient(ctx, includeMetadata), req)
	}
}

func enhanceStreamWithClientInformation(includeMetadata bool) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, wrapServerStream(contextWithClient(ss.Context(), includeMetadata), ss))
	}
}

// contextWithClient attempts to add the peer address to the client.Info from the context. When no
// client.Info exists in the context, one is created.
func contextWithClient(ctx context.Context, includeMetadata bool) context.Context {
	cl := client.FromContext(ctx)
	if p, ok := peer.FromContext(ctx); ok {
		cl.Addr = p.Addr
	}
	if includeMetadata {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			copiedMD := md.Copy()
			if len(md[client.MetadataHostName]) == 0 && len(md[":authority"]) > 0 {
				copiedMD[client.MetadataHostName] = md[":authority"]
			}
			cl.Metadata = client.NewMetadata(copiedMD)
		}
	}
	return client.NewContext(ctx, cl)
}

func authUnaryServerInterceptor(server extensionauth.Server) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		headers, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errMetadataNotFound
		}

		ctx, err := server.Authenticate(ctx, headers)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				return nil, s.Err()
			}

			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		return handler(ctx, req)
	}
}

func authStreamServerInterceptor(server extensionauth.Server) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		headers, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return errMetadataNotFound
		}

		ctx, err := server.Authenticate(ctx, headers)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				return s.Err()
			}

			return status.Error(codes.Unauthenticated, err.Error())
		}

		return handler(srv, wrapServerStream(ctx, stream))
	}
}
