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

package sender

import (
	"context"

	reporter "github.com/jaegertracing/jaeger/cmd/agent/app/reporter"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/errors/errorkind"
	jaegertranslator "github.com/open-telemetry/opentelemetry-service/translator/trace/jaeger"
)

// JaegerThriftTChannelSender takes span batches and sends them
// out on tchannel in thrift encoding
type JaegerThriftTChannelSender struct {
	logger   *zap.Logger
	reporter reporter.Reporter
}

var _ consumer.TraceConsumer = (*JaegerThriftTChannelSender)(nil)

// NewJaegerThriftTChannelSender creates new TChannel-based sender.
func NewJaegerThriftTChannelSender(
	reporter reporter.Reporter,
	zlogger *zap.Logger,
) *JaegerThriftTChannelSender {
	return &JaegerThriftTChannelSender{
		logger:   zlogger,
		reporter: reporter,
	}
}

// ConsumeTraceData sends the received data to the configured Jaeger Thrift end-point.
func (s *JaegerThriftTChannelSender) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	// TODO: (@pjanotti) In case of failure the translation to Jaeger Thrift is going to be remade, cache it somehow.
	tBatch, err := jaegertranslator.OCProtoToJaegerThrift(td)
	if err != nil {
		return errorkind.Permanent(err)
	}

	if err := s.reporter.EmitBatch(tBatch); err != nil {
		s.logger.Error("Reporter failed to report span batch", zap.Error(err))
		return err
	}
	return nil
}
