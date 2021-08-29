// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package skywalkingexporter

import (
	"context"
	"errors"
	"fmt"
	logpb "skywalking.apache.org/repo/goapi/collect/logging/v3"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/model/pdata"
)

// See https://godoc.org/google.golang.org/grpc#ClientConn.NewStream
// why we need to keep the cancel func to cancel the stream
type logsClientWithCancel struct {
	cancel context.CancelFunc
	tsec   logpb.LogReportService_CollectClient
}

type swExporter struct {
	cfg *Config
	// gRPC clients and connection.
	logSvcClient logpb.LogReportServiceClient
	// In any of the channels we keep always NumWorkers object (sometimes nil),
	// to make sure we don't open more than NumWorkers RPCs at any moment.
	logsClients    chan *logsClientWithCancel
	grpcClientConn *grpc.ClientConn
	metadata       metadata.MD
}

func newSwExporter(_ context.Context, cfg *Config) (*swExporter, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("Skywalking exporter cfg requires an Endpoint")
	}

	oce := &swExporter{
		cfg:      cfg,
		metadata: metadata.New(cfg.GRPCClientSettings.Headers),
	}
	return oce, nil
}

// start creates the gRPC client Connection
func (oce *swExporter) start(ctx context.Context, host component.Host) error {
	dialOpts, err := oce.cfg.GRPCClientSettings.ToDialOptions(host.GetExtensions())
	if err != nil {
		return err
	}
	var clientConn *grpc.ClientConn
	if clientConn, err = grpc.DialContext(ctx, oce.cfg.GRPCClientSettings.Endpoint, dialOpts...); err != nil {
		return err
	}

	oce.grpcClientConn = clientConn

	if oce.logsClients != nil {
		oce.logSvcClient = logpb.NewLogReportServiceClient(oce.grpcClientConn)
		// Try to create rpc clients now.
		for i := 0; i < oce.cfg.NumWorkers; i++ {
			// Populate the channel with NumWorkers nil RPCs to keep the number of workers
			// constant in the channel.
			oce.logsClients <- nil
		}
	}
	return nil
}

func (oce *swExporter) shutdown(context.Context) error {
	if oce.logsClients != nil {
		// First remove all the clients from the channel.
		for i := 0; i < oce.cfg.NumWorkers; i++ {
			<-oce.logsClients
		}
		// Now close the channel
		close(oce.logsClients)
	}
	return oce.grpcClientConn.Close()
}

func newExporter(ctx context.Context, cfg *Config) (*swExporter, error) {
	oce, err := newSwExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}
	oce.logsClients = make(chan *logsClientWithCancel, oce.cfg.NumWorkers)
	return oce, nil
}

func (oce *swExporter) pushLogs(_ context.Context, td pdata.Logs) error {
	// Get first available trace Client.
	tClient, ok := <-oce.logsClients
	if !ok {
		err := errors.New("failed to push traces, OpenCensus exporter was already stopped")
		return err
	}

	// In any of the metricsClients channel we keep always NumWorkers object (sometimes nil),
	// to make sure we don't open more than NumWorkers RPCs at any moment.
	// Here check if the client is nil and create a new one if that is the case. A nil
	// object means that an error happened: could not connect, service went down, etc.
	if tClient == nil {
		var err error
		tClient, err = oce.createLogServiceRPC()
		if err != nil {
			// Cannot create an RPC, put back nil to keep the number of workers constant.
			oce.logsClients <- nil
			return err
		}
	}

	for _, logData := range logDataToLogRecode(td) {
		err := tClient.tsec.Send(logData)
		if err != nil {
			return fmt.Errorf("failed to push log data via skywalking exporter: %w", err)
		}
	}

	oce.logsClients <- tClient
	return nil
}

func (oce *swExporter) createLogServiceRPC() (*logsClientWithCancel, error) {
	// Initiate the trace service by sending over node identifier info.
	ctx, cancel := context.WithCancel(context.Background())
	if len(oce.cfg.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(oce.cfg.Headers))
	}
	// Cannot use grpc.WaitForReady(cfg.WaitForReady) because will block forever.
	logClient, err := oce.logSvcClient.Collect(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("TraceServiceClient: %w", err)
	}
	return &logsClientWithCancel{cancel: cancel, tsec: logClient}, nil
}
