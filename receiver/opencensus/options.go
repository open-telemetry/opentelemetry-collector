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

package opencensus

import (
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/ocmetrics"
	"github.com/census-instrumentation/opencensus-service/receiver/opencensus/octrace"
)

// Option interface defines for configuration settings to be applied to receivers.
//
// withReceiver applies the configuration to the given receiver.
type Option interface {
	withReceiver(*Receiver)
}

type traceReceiverOptions struct {
	opts []octrace.Option
}

var _ Option = (*traceReceiverOptions)(nil)

func (tro *traceReceiverOptions) withReceiver(ocr *Receiver) {
	ocr.traceReceiverOpts = tro.opts
}

// WithTraceReceiverOptions is an option to specify the options that will be
// passed to the New call for octrace.Receiver
func WithTraceReceiverOptions(opts ...octrace.Option) Option {
	return &traceReceiverOptions{opts: opts}
}

type metricsReceiverOptions struct {
	opts []ocmetrics.Option
}

var _ Option = (*metricsReceiverOptions)(nil)

func (mro *metricsReceiverOptions) withReceiver(ocr *Receiver) {
	ocr.metricsReceiverOpts = mro.opts
}

// WithMetricsReceiverOptions is an option to specify the options that will be
// passed to the New call for ocmetrics.Receiver
func WithMetricsReceiverOptions(opts ...ocmetrics.Option) Option {
	return &metricsReceiverOptions{opts: opts}
}

type corsOrigins struct {
	origins []string
}

var _ Option = (*corsOrigins)(nil)

func (co *corsOrigins) withReceiver(ocr *Receiver) {
	ocr.corsOrigins = co.origins
}

// WithCorsOrigins is an option to specify the allowed origins to enable writing
// HTTP/JSON requests to the grpc-gateway adapter using CORS.
func WithCorsOrigins(origins []string) Option {
	return &corsOrigins{origins: origins}
}
