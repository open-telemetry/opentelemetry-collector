// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"context"
	"errors"
	"net"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/internal/sharedcomponent"
)

type registerServerFunc func(srv *grpc.Server)

type grpcComponent struct {
	cfg        *configgrpc.ServerConfig
	serverGRPC *grpc.Server
	tel        component.TelemetrySettings
	shutdownWG sync.WaitGroup
	servers    []registerServerFunc
}

func newSharedGRPC(cfg *configgrpc.ServerConfig, tel component.TelemetrySettings) (*sharedcomponent.Component[*grpcComponent], error) {
	return grpcs.LoadOrStore(
		cfg,
		func() (*grpcComponent, error) {
			return &grpcComponent{cfg: cfg, tel: tel}, nil
		},
	)
}

func (r *grpcComponent) Start(_ context.Context, host component.Host) error {
	var err error
	if r.serverGRPC, err = r.cfg.ToServer(context.Background(), host, r.tel); err != nil {
		return err
	}

	for _, registration := range r.servers {
		registration(r.serverGRPC)
	}

	r.tel.Logger.Info("Starting GRPC server", zap.String("endpoint", r.cfg.NetAddr.Endpoint))
	var gln net.Listener
	if gln, err = r.cfg.NetAddr.Listen(context.Background()); err != nil {
		return err
	}

	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()
		if errGrpc := r.serverGRPC.Serve(gln); errGrpc != nil && !errors.Is(errGrpc, grpc.ErrServerStopped) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errGrpc))
		}
	}()
	return nil
}

// Shutdown is a method to turn off receiving.
func (r *grpcComponent) Shutdown(context.Context) error {
	if r.serverGRPC != nil {
		r.serverGRPC.GracefulStop()
	}
	r.shutdownWG.Wait()
	return nil
}

func (r *grpcComponent) registerServer(sr registerServerFunc) {
	r.servers = append(r.servers, sr)
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the receiver.Factory is asked trace and metric receivers separately
// when it gets CreateTraces() and CreateMetrics() but they must not
// create separate objects, they must use one otlpReceiver object per configuration.
// When the receiver is shutdown it should be removed from this map so the same configuration
// can be recreated successfully.
var grpcs = sharedcomponent.NewMap[*configgrpc.ServerConfig, *grpcComponent]()
