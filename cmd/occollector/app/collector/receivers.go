// Copyright 2019, OpenCensus Authors
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

package collector

import (
	"context"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	jaegerreceiver "github.com/census-instrumentation/opencensus-service/internal/collector/jaeger"
	ocreceiver "github.com/census-instrumentation/opencensus-service/internal/collector/opencensus"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	zipkinreceiver "github.com/census-instrumentation/opencensus-service/internal/collector/zipkin"
	zipkinscribereceiver "github.com/census-instrumentation/opencensus-service/internal/collector/zipkin/scribe"
	"github.com/census-instrumentation/opencensus-service/receiver"
)

func createReceivers(v *viper.Viper, logger *zap.Logger, spanProcessor processor.SpanProcessor) []receiver.TraceReceiver {
	var someReceiverEnabled bool
	receivers := []struct {
		name    string
		runFn   func(*zap.Logger, *viper.Viper, processor.SpanProcessor) (receiver.TraceReceiver, error)
		enabled bool
	}{
		{"Jaeger", jaegerreceiver.Start, builder.JaegerReceiverEnabled(v)},
		{"OpenCensus", ocreceiver.Start, builder.OpenCensusReceiverEnabled(v)},
		{"Zipkin", zipkinreceiver.Start, builder.ZipkinReceiverEnabled(v)},
		{"Zipkin-Scribe", zipkinscribereceiver.Start, builder.ZipkinScribeReceiverEnabled(v)},
	}

	var startedTraceReceivers []receiver.TraceReceiver
	for _, receiver := range receivers {
		if receiver.enabled {
			rec, err := receiver.runFn(logger, v, spanProcessor)
			if err != nil {
				// TODO: (@pjanotti) better shutdown, for now just try to stop any started receiver before terminating.
				for _, startedTraceReceiver := range startedTraceReceivers {
					startedTraceReceiver.StopTraceReception(context.Background())
				}
				logger.Fatal("Cannot run receiver for "+receiver.name, zap.Error(err))
			}
			startedTraceReceivers = append(startedTraceReceivers, rec)
			someReceiverEnabled = true
		}
	}

	if !someReceiverEnabled {
		logger.Warn("Nothing to do: no receiver was enabled. Shutting down.")
		os.Exit(1)
	}

	return startedTraceReceivers
}
