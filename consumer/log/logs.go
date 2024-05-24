// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package log // import "go.opentelemetry.io/collector/consumer/log"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/internal/base"
	"go.opentelemetry.io/collector/pdata/plog"
)

var errNilFunc = errors.New("nil consumer func")

type config struct {
	baseOptions []base.Option
}

// Option to construct new consumers.
type Option func(config) config

// Logs is an interface that receives plog.Logs, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Logs interface {
	base.Consumer
	// ConsumeLogs receives plog.Logs for consumption.
	ConsumeLogs(ctx context.Context, ld plog.Logs) error
}

// ConsumeLogsFunc is a helper function that is similar to ConsumeLogs.
type ConsumeLogsFunc func(ctx context.Context, ld plog.Logs) error

// ConsumeLogs calls f(ctx, ld).
func (f ConsumeLogsFunc) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return f(ctx, ld)
}

type baseLogs struct {
	*base.Impl
	ConsumeLogsFunc
}

// NewLogs returns a Logs configured with the provided options.
func NewLogs(consume ConsumeLogsFunc, options ...Option) (Logs, error) {
	if consume == nil {
		return nil, errNilFunc
	}

	cfg := config{}
	for _, op := range options {
		cfg = op(cfg)
	}

	return &baseLogs{
		Impl:            base.NewImpl(cfg.baseOptions...),
		ConsumeLogsFunc: consume,
	}, nil
}

// WithCapabilities overrides the default GetCapabilities function for a processor.
// The default GetCapabilities function returns mutable capabilities.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(c config) config {
		c.baseOptions = append(c.baseOptions, base.WithCapabilities(capabilities))
		return c
	}
}
