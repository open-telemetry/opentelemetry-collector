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

package kafkaexporter

import (
	"context"
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestNewExporter_wrong_version(t *testing.T) {
	c := Config{ProtocolVersion: "0.0.0"}
	exp, err := newExporter(c, component.ExporterCreateParams{})
	assert.Error(t, err)
	assert.Nil(t, exp)
}

func TestTraceDataPusher(t *testing.T) {
	c := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, c)
	producer.ExpectSendMessageAndSucceed()

	p := kafkaProducer{
		producer:   producer,
		marshaller: &protoMarshaller{},
	}
	t.Cleanup(func() {
		require.NoError(t, p.Close(context.Background()))
	})
	droppedSpans, err := p.traceDataPusher(context.Background(), testdata.GenerateTraceDataTwoSpansSameResource())
	require.NoError(t, err)
	assert.Equal(t, 0, droppedSpans)
}

func TestTraceDataPusher_err(t *testing.T) {
	c := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, c)
	expErr := fmt.Errorf("failed to send")
	producer.ExpectSendMessageAndFail(expErr)

	p := kafkaProducer{
		producer:   producer,
		marshaller: &protoMarshaller{},
		logger:     zap.NewNop(),
	}
	t.Cleanup(func() {
		require.NoError(t, p.Close(context.Background()))
	})
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	droppedSpans, err := p.traceDataPusher(context.Background(), td)
	assert.EqualError(t, err, expErr.Error())
	assert.Equal(t, td.SpanCount(), droppedSpans)
}

func TestTraceDataPusher_marshall_error(t *testing.T) {
	expErr := fmt.Errorf("failed to marshall")
	p := kafkaProducer{
		marshaller: &errorMarshaller{err: expErr},
		logger:     zap.NewNop(),
	}
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	droppedSpans, err := p.traceDataPusher(context.Background(), td)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expErr.Error())
	assert.Equal(t, td.SpanCount(), droppedSpans)
}

type errorMarshaller struct {
	err error
}

var _ marshaller = (*errorMarshaller)(nil)

func (e errorMarshaller) Marshal(pdata.Traces) ([]byte, error) {
	return nil, e.err
}
