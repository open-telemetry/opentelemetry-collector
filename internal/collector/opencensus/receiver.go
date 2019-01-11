// Copyright 2018, OpenCensus Authors
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
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/octrace"
)

// Run starts the OpenCensus receiver endpoint.
func Run(logger *zap.Logger, v *viper.Viper, spanProc processor.SpanProcessor) (func(), error) {
	rOpts, err := builder.NewDefaultOpenCensusReceiverCfg().InitFromViper(v)
	if err != nil {
		return nil, err
	}

	grpcSrv := internal.GRPCServerWithObservabilityEnabled()

	// Temporarily disabling the grpc metrics since they do not provide good data at this moment,
	// See https://github.com/census-instrumentation/opencensus-service/issues/287
	// if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
	// 	return nil, fmt.Errorf("Failed to register ocgrpc.DefaultServerViews: %v", err)
	// }

	lis, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(rOpts.Port), 10))
	if err != nil {
		return nil, fmt.Errorf("Cannot bind tcp listener to address: %v", err)
	}

	ss := processor.WrapWithSpanSink("oc", spanProc)
	oci, err := octrace.New(ss, octrace.WithSpanBufferPeriod(1*time.Second))
	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus receiver: %v", err)
	}

	agenttracepb.RegisterTraceServiceServer(grpcSrv, oci)
	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("OpenCensus gRPC shutdown", zap.Error(err))
		}
	}()

	logger.Info("OpenCensus receiver is running.", zap.Int("port", rOpts.Port))

	return grpcSrv.Stop, nil
}
