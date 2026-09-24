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
	"net/url"
	"strconv"
	"strings"

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
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configgrpc/internal/grpccompression/snappy"
	"go.opentelemetry.io/collector/config/configgrpc/internal/grpccompression/zstd"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension/extensionauth"
)

// DefaultBalancerName is the name of the default load balancer.
const DefaultBalancerName = "round_robin"

var errMetadataNotFound = errors.New("no request metadata found")

var (
	_ confmap.Validator = (*ClientConfig)(nil)
	_ confmap.Validator = (*ServerConfig)(nil)
)

func validateClientConfig(cc *ClientConfig) error {
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

// sanitizedEndpoint strips the URI scheme and authority from the endpoint to
// extract the host:port for validation. It handles http://, https://, and any
// gRPC resolver scheme URI (e.g. dns:///host:port, passthrough:///host:port).
// For gRPC URIs of the form "scheme://[authority]/endpoint", the authority is
// also stripped, matching the parsing behavior of grpc-go's url.Parse approach.
func (cc *ClientConfig) sanitizedEndpoint() string {
	switch {
	case cc.isSchemeHTTP():
		return strings.TrimPrefix(cc.Endpoint, "http://")
	case cc.isSchemeHTTPS():
		return strings.TrimPrefix(cc.Endpoint, "https://")
	default:
		// Only attempt URI parsing if the endpoint contains "://", which
		// distinguishes a scheme URI (e.g. "dns:///host:port") from a bare
		// host:port. Without this check, url.Parse("host:port") would
		// misinterpret "host" as the scheme.
		if !strings.Contains(cc.Endpoint, "://") {
			return cc.Endpoint
		}
		// Parse as a URI to strip scheme and authority, matching how grpc-go
		// parses target URIs via url.Parse in grpc.NewClient.
		u, err := url.Parse(cc.Endpoint)
		if err != nil {
			return cc.Endpoint
		}
		return strings.TrimPrefix(u.Path, "/")
	}
}

// grpcDialTarget returns the target string to pass to grpc.NewClient.
// For http:// and https:// prefixes (which are not gRPC resolver schemes),
// the prefix is stripped. For all other endpoints, the value is passed through
// to grpc.NewClient as-is, allowing any gRPC resolver scheme (e.g. dns:///,
// passthrough:///, xds:///) to be used directly.
func (cc *ClientConfig) grpcDialTarget() string {
	switch {
	case cc.isSchemeHTTP(), cc.isSchemeHTTPS():
		return cc.sanitizedEndpoint()
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
	if cc.Dialer.HasValue() {
		fn, rerr := cc.Dialer.Get().GetDialer(ctx, extensions)
		if rerr != nil {
			return nil, rerr
		}
		grpcOpts = append(grpcOpts, grpc.WithContextDialer(func(ctx context.Context, address string) (net.Conn, error) {
			return fn(ctx, "tcp", address)
		}))
	}
	conn, err := grpc.NewClient(cc.grpcDialTarget(), grpcOpts...)
	if err != nil {
		return nil, err
	}

	// Initiate connection to match the previous behavior of DialContext
	// This ensures the connection is established eagerly rather than lazily
	conn.Connect()

	return conn, nil
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
	var callOpts []grpc.CallOption
	callOpts = append(callOpts, grpc.WaitForReady(cc.WaitForReady))
	if cc.Compression.IsCompressed() {
		cp, err := getGRPCCompressionName(cc.Compression)
		if err != nil {
			return nil, err
		}
		callOpts = append(callOpts, grpc.UseCompressor(cp))
	}
	opts = append(opts, grpc.WithDefaultCallOptions(callOpts...))

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
			return nil, perr
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

	if cc.UserAgent != "" {
		opts = append(opts, grpc.WithUserAgent(cc.UserAgent))
	}

	return opts, nil
}

func validateMaxRecvMsgSizeMiB(maxRecvMsgSizeMiB int) error {
	if maxRecvMsgSizeMiB*1024*1024 < 0 {
		return fmt.Errorf("invalid max_recv_msg_size_mib value, must be between 1 and %d: %d", math.MaxInt/1024/1024, maxRecvMsgSizeMiB)
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

	// Add client info first, before auth.
	uInterceptors = append(uInterceptors, enhanceWithClientInformation(sc.IncludeMetadata))
	sInterceptors = append(sInterceptors, enhanceStreamWithClientInformation(sc.IncludeMetadata)) //nolint:contextcheck // context already handled

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
