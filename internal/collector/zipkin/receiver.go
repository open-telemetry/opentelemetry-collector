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

// Package zipkinreceiver wraps the functionality to start the end-point that
// receives Zipkin traces.
package zipkinreceiver

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	zr "github.com/census-instrumentation/opencensus-service/receiver/zipkin"
)

// Run starts the Zipkin receiver endpoint.
func Run(logger *zap.Logger, v *viper.Viper, spanProc processor.SpanProcessor) (func(), error) {
	rOpts, err := builder.NewDefaultZipkinReceiverCfg().InitFromViper(v)
	if err != nil {
		return nil, err
	}

	// TODO: (@pjanotti) when Zipkin implementation of StartTraceReceiver is working, change this code (this temporarily).
	ss := processor.WrapWithSpanSink("zipkin", spanProc)
	addr := ":" + strconv.FormatInt(int64(rOpts.Port), 10)
	zi, err := zr.New(ss)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the Zipkin receiver: %v", err)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("Cannot bind Zipkin receiver to address %q: %v", addr, err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v2/spans", zi)
	go func() {
		if err := http.Serve(ln, mux); err != nil {
			logger.Fatal("Failed to serve the Zipkin receiver: %v", zap.Error(err))
		}
	}()

	logger.Info("Zipkin receiver is running.", zap.Int("port", rOpts.Port))

	doneFn := func() {
		ln.Close()
	}
	return doneFn, nil
}
