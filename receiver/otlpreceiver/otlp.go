// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/logs"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metrics"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/trace"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

// baseReceiver is the type that exposes Trace and Metrics reception.
type baseReceiver struct {
	comps []component.Component
}

func (r *baseReceiver) Start(ctx context.Context, host component.Host) error {
	for _, comp := range r.comps {
		if err := comp.Start(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

func (r *baseReceiver) Shutdown(ctx context.Context) error {
	var errs []error
	for _, comp := range r.comps {
		if err := comp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func newTraces(set receiver.Settings, cfg *Config, next consumer.Traces) (receiver.Traces, error) {
	br := &baseReceiver{}
	if cfg.GRPC != nil {
		serv, err := newSharedGRPC(cfg.GRPC, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := trace.New(next, set, transportGRPC)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerServer(func(srv *grpc.Server) {
			ptraceotlp.RegisterGRPCServer(srv, recv)
		})
	}
	if cfg.HTTP != nil {
		serv, err := newSharedHTTP(cfg.HTTP.ServerConfig, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := trace.New(next, set, transportHTTP)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerHandler(cfg.HTTP.TracesURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleTraces(resp, req, recv)
		})
	}
	return br, nil
}

func newMetrics(set receiver.Settings, cfg *Config, next consumer.Metrics) (receiver.Metrics, error) {
	br := &baseReceiver{}
	if cfg.GRPC != nil {
		serv, err := newSharedGRPC(cfg.GRPC, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := metrics.New(next, set, transportGRPC)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerServer(func(srv *grpc.Server) {
			pmetricotlp.RegisterGRPCServer(srv, recv)
		})
	}
	if cfg.HTTP != nil {
		serv, err := newSharedHTTP(cfg.HTTP.ServerConfig, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := metrics.New(next, set, transportHTTP)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerHandler(cfg.HTTP.MetricsURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleMetrics(resp, req, recv)
		})
	}
	return br, nil
}

func newLogs(set receiver.Settings, cfg *Config, next consumer.Logs) (receiver.Logs, error) {
	br := &baseReceiver{}
	if cfg.GRPC != nil {
		serv, err := newSharedGRPC(cfg.GRPC, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := logs.New(next, set, transportGRPC)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerServer(func(srv *grpc.Server) {
			plogotlp.RegisterGRPCServer(srv, recv)
		})
	}
	if cfg.HTTP != nil {
		serv, err := newSharedHTTP(cfg.HTTP.ServerConfig, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv, err := logs.New(next, set, transportHTTP)
		if err != nil {
			return nil, err
		}
		serv.Unwrap().registerHandler(cfg.HTTP.LogsURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleLogs(resp, req, recv)
		})
	}
	return br, nil
}

func newProfiles(set receiver.Settings, cfg *Config, next xconsumer.Profiles) (xreceiver.Profiles, error) {
	br := &baseReceiver{}
	if cfg.GRPC != nil {
		serv, err := newSharedGRPC(cfg.GRPC, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv := profiles.New(next)
		serv.Unwrap().registerServer(func(srv *grpc.Server) {
			pprofileotlp.RegisterGRPCServer(srv, recv)
		})
	}
	if cfg.HTTP != nil {
		serv, err := newSharedHTTP(cfg.HTTP.ServerConfig, set.TelemetrySettings)
		if err != nil {
			return nil, err
		}
		br.comps = append(br.comps, serv)
		recv := profiles.New(next)
		serv.Unwrap().registerHandler(defaultProfilesURLPath, func(resp http.ResponseWriter, req *http.Request) {
			handleProfiles(resp, req, recv)
		})
	}
	return br, nil
}
