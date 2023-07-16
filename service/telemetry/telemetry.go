// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"runtime"

	"github.com/google/uuid"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configrotate"
)

type Telemetry struct {
	logger         *zap.Logger
	tracerProvider *sdktrace.TracerProvider
}

func (t *Telemetry) TracerProvider() trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) Logger() *zap.Logger {
	return t.logger
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	// TODO: Sync logger.
	return multierr.Combine(
		t.tracerProvider.Shutdown(ctx),
	)
}

// Settings holds configuration for building Telemetry.
type Settings struct {
	ZapOptions []zap.Option
}

// New creates a new Telemetry from Config.
func New(_ context.Context, set Settings, cfg Config) (*Telemetry, error) {
	logger, err := newLogger(cfg.Logs, set.ZapOptions)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		// needed for supporting the zpages extension
		sdktrace.WithSampler(alwaysRecord()),
	)
	return &Telemetry{
		logger:         logger,
		tracerProvider: tp,
	}, nil
}

func newLogger(cfg LogsConfig, options []zap.Option) (*zap.Logger, error) {
	// Copied from NewProductionConfig.
	zapCfg := &zap.Config{
		Level:             zap.NewAtomicLevelAt(cfg.Level),
		Development:       cfg.Development,
		Sampling:          toSamplingConfig(cfg.Sampling),
		Encoding:          cfg.Encoding,
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       cfg.OutputPaths,
		ErrorOutputPaths:  cfg.ErrorOutputPaths,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		InitialFields:     cfg.InitialFields,
	}

	if zapCfg.Encoding == "console" {
		// Human-readable timestamps for console format of logs.
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	if cfg.Rotation != nil && cfg.Rotation.Enabled {
		rotationSchema := "rotation-" + uuid.NewString()
		err := zap.RegisterSink(rotationSchema, getRotationSinkFactory(cfg.Rotation))
		if err != nil {
			return nil, err
		}
		zapCfg.OutputPaths, err = setRotationURL(zapCfg.OutputPaths, rotationSchema)
		if err != nil {
			return nil, err
		}
		zapCfg.ErrorOutputPaths, err = setRotationURL(zapCfg.ErrorOutputPaths, rotationSchema)
		if err != nil {
			return nil, err
		}
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func toSamplingConfig(sc *LogsSamplingConfig) *zap.SamplingConfig {
	if sc == nil {
		return nil
	}
	return &zap.SamplingConfig{
		Initial:    sc.Initial,
		Thereafter: sc.Thereafter,
	}
}

func getRotationSinkFactory(cfg *configrotate.Config) func(u *url.URL) (zap.Sink, error) {
	return func(u *url.URL) (zap.Sink, error) {
		p := u.Query().Get("path")
		writer, err := cfg.NewWriter(p)
		if err != nil {
			return nil, err
		}
		return nopSyncSink{writer}, nil
	}
}

// lumberjack.Logger does not provide a Sync() method, which is required by zap.Sink
// explanation: https://github.com/natefinch/lumberjack/pull/47#issuecomment-322502210
type nopSyncSink struct {
	io.WriteCloser
}

func (w nopSyncSink) Sync() error {
	return nil
}

func setRotationURL(paths []string, rotationSchema string) ([]string, error) {
	res := make([]string, 0, len(paths))
	for _, p := range paths {
		if runtime.GOOS == "windows" && filepath.IsAbs(p) {
			res = append(res, rotationSchema+":?path="+url.QueryEscape(p))
			continue
		}
		u, err := url.Parse(p)
		if err != nil {
			return nil, err
		}
		if (u.Scheme == "" || u.Scheme == "file") &&
			u.Path != "stdout" && u.Path != "stderr" {
			// Copied from zap. Only clean URLs are allowed
			if u.User != nil {
				return nil, fmt.Errorf("user and password not allowed with file URLs: got %v", u)
			}
			if u.Fragment != "" {
				return nil, fmt.Errorf("fragments not allowed with file URLs: got %v", u)
			}
			if u.RawQuery != "" {
				return nil, fmt.Errorf("query parameters not allowed with file URLs: got %v", u)
			}
			// Error messages are better if we check hostname and port separately.
			if u.Port() != "" {
				return nil, fmt.Errorf("ports not allowed with file URLs: got %v", u)
			}
			if hn := u.Hostname(); hn != "" && hn != "localhost" {
				return nil, fmt.Errorf("file URLs must leave host empty or use localhost: got %v", u)
			}

			res = append(res, rotationSchema+":?path="+url.QueryEscape(u.Path))
			continue
		}
		res = append(res, p)
	}
	return res, nil
}
