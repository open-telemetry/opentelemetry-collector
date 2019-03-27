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
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/receiver"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensusreceiver"
)

// Start starts the OpenCensus receiver endpoint.
func Start(logger *zap.Logger, v *viper.Viper, traceConsumer consumer.TraceConsumer, asyncErrorChan chan<- error) (receiver.TraceReceiver, error) {
	rOpts, err := builder.NewDefaultOpenCensusReceiverCfg().InitFromViper(v)
	if err != nil {
		return nil, err
	}

	tlsCredsOption, hasTLSCreds, err := rOpts.TLSCredentials.ToOpenCensusReceiverServerOption()
	if err != nil {
		return nil, fmt.Errorf("OpenCensus receiver TLS Credentials: %v", err)
	}

	addr := ":" + strconv.FormatInt(int64(rOpts.Port), 10)
	ocr, err := opencensusreceiver.New(addr, traceConsumer, nil, tlsCredsOption)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the OpenCensus trace receiver: %v", err)
	}

	if err := ocr.StartTraceReception(context.Background(), asyncErrorChan); err != nil {
		return nil, fmt.Errorf("Cannot bind Opencensus receiver to address %q: %v", addr, err)
	}

	if hasTLSCreds {
		tlsCreds := rOpts.TLSCredentials
		logger.Info("OpenCensus receiver is running.",
			zap.Int("port", rOpts.Port),
			zap.String("cert_file", tlsCreds.CertFile),
			zap.String("key_file", tlsCreds.KeyFile))
	} else {
		logger.Info("OpenCensus receiver is running.", zap.Int("port", rOpts.Port))
	}

	return ocr, nil
}
