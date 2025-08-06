# Optional[T] type for use in configuration structs

## Overview

In Go, types are by default set to a "zero-value", a supported value for the
respective type that is semantically equivalent or similar to "empty", but which
is also a valid value for the type. For many config fields, the zero value is
not a valid configuration value and can be taken to mean that the option is
disabled, but in certain cases it can indicate a default value, necessitating a
way to represent the presence of a value without using a valid value for the
type.

Using standard Go without inventing any new types, the two most straightforward
ways to accomplish this are:

1. Using a separate boolean field to indicate whether the field is enabled or
   disabled.
2. Making the type a pointer, which makes a `nil` pointer represent that a
   value is not present, and a valid pointer represent the desired value.

Each of these approaches has deficiencies: Using a separate boolean field
requires the user to set the boolean field to `true` in addition to setting the
actual config option, leading to suboptimal UX. Using a pointer value has a few
drawbacks:

1. It may not be immediately obvious to a new user that a pointer type indicates
   a field is optional.
2. The distinction between values that are conventionally pointers (e.g. gRPC
   configs) and optional values is lost.
3. Setting a default value for a pointer field when decoding will set the field
   on the resulting config struct, and additional logic must be done to unset
   the default if the user has not specified a value.
4. The potential for null pointer exceptions is created.
5. Config structs are generally intended to be immutable and may be passed
   around a lot, which makes the mutability property of pointer fields
   an undesirable property.

## Optional types

Go does not include any form of Optional type in the standard library, but other
popular languages like Rust and Java do. We could implement something similar in
our config packages that allows us to address the downsides of using pointers
for optional config fields.

## Basic definition

A production-grade implementation will not be discussed or shown here, but the
basic implementation for an Optional type in Go could look something like:

```golang
type Optional[T any] struct {
	hasValue bool
	value    T

	defaultVal T
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func WithDefault[T any](defaultVal T) Optional[T] {
	return Optional[T]{defaultVal: defaultVal}
}

func (o Optional[T]) HasValue() bool {
	return o.hasValue
}

func (o Optional[T]) Value() T {
    return o.value
}

func (o Optional[T]) Default() T {
    return o.defaultVal
}
```

## Use cases

Optional types can fulfill the following use cases we have when decoding config.

### Representing optional config fields

To use the optional type to mark a config field as optional, the type can simply be used as a type
parameter to `Optional[T]`. The YAML representation of `Optional[T]` is the same as that of `T`,
except that the type records whether the value is present. The following config struct shows how
this may look, both in definition and in usage:

```golang
type Protocols struct {
	GRPC Optional[configgrpc.ServerConfig] `mapstructure:"grpc"`
	HTTP Optional[HTTPConfig]              `mapstructure:"http"`
}

func (cfg *Config) Validate() error {
	if !cfg.GRPC.HasValue() && !cfg.HTTP.HasValue() {
		return errors.New("must specify at least one protocol when using the OTLP receiver")
	}
	return nil
}

func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: WithDefault(configgrpc.ServerConfig{
                // ...
            }),
			HTTP: WithDefault(HTTPConfig{
                // ...
            }),
		},
	}
}
```

For something like `confighttp.ServerConfig`, using an `Optional[T]` type for
optional fields would look like this:

```golang
type ServerConfig struct {
	TLSSetting Optional[configtls.ServerConfig] `mapstructure:"tls"`

	CORS Optional[CORSConfig] `mapstructure:"cors"`

	Auth Optional[AuthConfig] `mapstructure:"auth,omitempty"`

	ResponseHeaders Optional[map[string]configopaque.String] `mapstructure:"response_headers"`
}

func NewDefaultServerConfig() ServerConfig {
	return ServerConfig{
		TLSSetting:        WithDefault(configtls.NewDefaultServerConfig()),
		CORS:              WithDefault(NewDefaultCORSConfig()),
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 1 * time.Minute,
		IdleTimeout:       1 * time.Minute,
	}
}

func (sc *ServerConfig) ToListener(ctx context.Context) (net.Listener, error) {
	listener, err := net.Listen("tcp", sc.Endpoint)
	if err != nil {
		return nil, err
	}

	if sc.TLSSetting.HasValue() {
		var tlsCfg *tls.Config
		tlsCfg, err = sc.TLSSetting.Value().LoadTLSConfig(ctx)
		if err != nil {
			return nil, err
		}
		tlsCfg.NextProtos = []string{http2.NextProtoTLS, "http/1.1"}
		listener = tls.NewListener(listener, tlsCfg)
	}

	return listener, nil
}

func (sc *ServerConfig) ToServer(_ context.Context, host component.Host, settings component.TelemetrySettings, handler http.Handler, opts ...ToServerOption) (*http.Server, error) {
	// ...

	handler = httpContentDecompressor(
		handler,
		sc.MaxRequestBodySize,
		serverOpts.ErrHandler,
		sc.CompressionAlgorithms,
		serverOpts.Decoders,
	)

	// ...

	if sc.Auth.HasValue() {
		server, err := sc.Auth.Value().GetServerAuthenticator(context.Background(), host.GetExtensions())
		if err != nil {
			return nil, err
		}

		handler = authInterceptor(handler, server, sc.Auth.Value().RequestParameters)
	}

	corsValue := sc.CORS.Value()
	if sc.CORS.HasValue() && len(sc.CORS.AllowedOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   corsValue.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   corsValue.AllowedHeaders,
			MaxAge:           corsValue.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}
	if sc.CORS.HasValue() && len(sc.CORS.AllowedOrigins) == 0 && len(sc.CORS.AllowedHeaders) > 0 {
		settings.Logger.Warn("The CORS configuration specifies allowed headers but no allowed origins, and is therefore ignored.")
	}

	if sc.ResponseHeaders.HasValue() {
		handler = responseHeadersHandler(handler, sc.ResponseHeaders.Value())
	}

	// ...
}
```

### Proper unmarshaling of empty values when a default is set

Currently, the OTLP receiver requires a workaround to make enabling each
protocol optional while providing defaults if a key for the protocol has been
set:

```golang
type Protocols struct {
	GRPC *configgrpc.ServerConfig `mapstructure:"grpc"`
	HTTP *HTTPConfig              `mapstructure:"http"`
}

// Config defines configuration for OTLP receiver.
type Config struct {
	// Protocols is the configuration for the supported protocols, currently gRPC and HTTP (Proto and JSON).
	Protocols `mapstructure:"protocols"`
}

func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: configgrpc.NewDefaultServerConfig(),
			HTTP: &HTTPConfig{
                // ...
			},
		},
	}
}

func (cfg *Config) Unmarshal(conf *confmap.Conf) error {
	// first load the config normally
	err := conf.Unmarshal(cfg)
	if err != nil {
		return err
	}

    // gRPC will be enabled if this line is not run
	if !conf.IsSet(protoGRPC) {
        cfg.GRPC = nil
	}

     // HTTP will be enabled if this line is not run
	if !conf.IsSet(protoHTTP) {
		cfg.HTTP = nil
	}

	return nil
}
```

With an Optional type, the checks in `Unmarshal` become unnecessary, and it's
possible the entire `Unmarshal` function may no longer be needed. Instead, when
the config is unmarshaled, no value would be put into the default Optional
values and `HasValue` would return false when using this config object.

This situation is something of an edge case with our current unmarshaling
facilities and while not common, could be a rough edge for users looking to
implement similar config structures.

## Disadvantages of an Optional type

There is one noteworthy disadvantage of introducing an Optional type:

1. Since the type isn't standard, external packages working with config may
   require additional adaptations to work with our config structs. For example,
   if we wanted to generate our types from a JSON schema using a package like
   [github.com/atombender/go-jsonschema][go-jsonschema], we would need some way
   to ensure compatibility with an Optional type.

[go-jsonschema]: https://github.com/omissis/go-jsonschema
