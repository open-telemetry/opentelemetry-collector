// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ocreceiver wraps the functionality to start the end-point that
// receives data directly in the OpenCensus format.
package ocreceiver

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/open-telemetry/opentelemetry-service/cmd/occollector/app/builder"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/opencensusreceiver"
)

// Start starts the OpenCensus receiver endpoint.
func Start(logger *zap.Logger, v *viper.Viper, traceConsumer consumer.TraceConsumer, asyncErrorChan chan<- error) (receiver.TraceReceiver, error) {
	addr, opts, zapFields, err := receiverOptions(v)
	if err != nil {
		return nil, err
	}

	ocr, err := opencensusreceiver.New(addr, traceConsumer, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus trace receiver: %v", err)
	}

	if err := ocr.StartTraceReception(context.Background(), asyncErrorChan); err != nil {
		return nil, fmt.Errorf("Cannot bind Opencensus receiver to address %q: %v", addr, err)
	}

	logger.Info("OpenCensus receiver is running.", zapFields...)

	return ocr, nil
}

func receiverOptions(v *viper.Viper) (addr string, opts []opencensusreceiver.Option, zapFields []zap.Field, err error) {
	rOpts, err := builder.NewDefaultOpenCensusReceiverCfg().InitFromViper(v)
	if err != nil {
		return addr, opts, zapFields, err
	}

	tlsCredsOption, hasTLSCreds, err := rOpts.TLSCredentials.ToOpenCensusReceiverServerOption()
	if err != nil {
		return addr, opts, zapFields, fmt.Errorf("OpenCensus receiver TLS Credentials: %v", err)
	}
	if hasTLSCreds {
		opts = append(opts, tlsCredsOption)
		tlsCreds := rOpts.TLSCredentials
		zapFields = append(zapFields, zap.String("cert_file", tlsCreds.CertFile), zap.String("key_file", tlsCreds.KeyFile))
	}

	grpcServerOptions, zapFields := grpcServerOptions(rOpts, zapFields)
	if len(grpcServerOptions) > 0 {
		opts = append(opts, opencensusreceiver.WithGRPCServerOptions(grpcServerOptions...))
	}

	addr = ":" + strconv.FormatInt(int64(rOpts.Port), 10)
	zapFields = append(zapFields, zap.Int("port", rOpts.Port))

	return addr, opts, zapFields, err
}

func grpcServerOptions(rOpts *builder.OpenCensusReceiverCfg, zapFields []zap.Field) ([]grpc.ServerOption, []zap.Field) {
	var grpcServerOptions []grpc.ServerOption
	if rOpts.MaxRecvMsgSizeMiB > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxRecvMsgSize(int(rOpts.MaxRecvMsgSizeMiB*1024*1024)))
		zapFields = append(zapFields, zap.Uint64("max-recv-msg-size-mib", rOpts.MaxRecvMsgSizeMiB))
	}
	if rOpts.MaxConcurrentStreams > 0 {
		grpcServerOptions = append(grpcServerOptions, grpc.MaxConcurrentStreams(rOpts.MaxConcurrentStreams))
		zapFields = append(zapFields, zap.Uint32("max-concurrent-streams", rOpts.MaxConcurrentStreams))
	}
	if rOpts.Keepalive != nil {
		if rOpts.Keepalive.ServerParameters != nil {
			svrParams := rOpts.Keepalive.ServerParameters
			grpcServerOptions = append(grpcServerOptions, grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     svrParams.MaxConnectionIdle,
				MaxConnectionAge:      svrParams.MaxConnectionAge,
				MaxConnectionAgeGrace: svrParams.MaxConnectionAgeGrace,
				Time:                  svrParams.Time,
				Timeout:               svrParams.Timeout,
			}))
			zapFields = append(zapFields, zap.Any("keepalive.server-parameters", rOpts.Keepalive.ServerParameters))
		}
		if rOpts.Keepalive.EnforcementPolicy != nil {
			enfPol := rOpts.Keepalive.EnforcementPolicy
			grpcServerOptions = append(grpcServerOptions, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             enfPol.MinTime,
				PermitWithoutStream: enfPol.PermitWithoutStream,
			}))
			zapFields = append(zapFields, zap.Any("keepalive.enforcement-policy", rOpts.Keepalive.EnforcementPolicy))
		}
	}

	return grpcServerOptions, zapFields
}
