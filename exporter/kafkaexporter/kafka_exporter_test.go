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
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestNewExporter_err_version(t *testing.T) {
	c := Config{ProtocolVersion: "0.0.0", Encoding: defaultEncoding}
	texp, err := newTracesExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, tracesMarshallers())
	assert.Error(t, err)
	assert.Nil(t, texp)
}

func TestNewExporter_err_encoding(t *testing.T) {
	c := Config{Encoding: "foo"}
	texp, err := newTracesExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, tracesMarshallers())
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
	assert.Nil(t, texp)
}

func TestNewMetricsExporter_err_version(t *testing.T) {
	c := Config{ProtocolVersion: "0.0.0", Encoding: defaultEncoding}
	mexp, err := newMetricsExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, metricsMarshallers())
	assert.Error(t, err)
	assert.Nil(t, mexp)
}

func TestNewMetricsExporter_err_encoding(t *testing.T) {
	c := Config{Encoding: "bar"}
	mexp, err := newMetricsExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, metricsMarshallers())
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
	assert.Nil(t, mexp)
}

func TestNewMetricsExporter_err_traces_encoding(t *testing.T) {
	c := Config{Encoding: "jaeger_proto"}
	mexp, err := newMetricsExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, metricsMarshallers())
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
	assert.Nil(t, mexp)
}

func TestNewExporter_err_auth_type(t *testing.T) {
	c := Config{
		ProtocolVersion: "2.0.0",
		Authentication: Authentication{
			TLS: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: "/doesnotexist",
				},
			},
		},
		Encoding: defaultEncoding,
		Metadata: Metadata{
			Full: false,
		},
	}
	texp, err := newTracesExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, tracesMarshallers())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load TLS config")
	assert.Nil(t, texp)
	mexp, err := newMetricsExporter(c, component.ExporterCreateParams{Logger: zap.NewNop()}, metricsMarshallers())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load TLS config")
	assert.Nil(t, mexp)
}

func TestTraceDataPusher(t *testing.T) {
	c := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, c)
	producer.ExpectSendMessageAndSucceed()

	p := kafkaTracesProducer{
		producer:   producer,
		marshaller: &otlpTracesPbMarshaller{},
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

	p := kafkaTracesProducer{
		producer:   producer,
		marshaller: &otlpTracesPbMarshaller{},
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
	p := kafkaTracesProducer{
		marshaller: &tracesErrorMarshaller{err: expErr},
		logger:     zap.NewNop(),
	}
	td := testdata.GenerateTraceDataTwoSpansSameResource()
	droppedSpans, err := p.traceDataPusher(context.Background(), td)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expErr.Error())
	assert.Equal(t, td.SpanCount(), droppedSpans)
}

func TestMetricsDataPusher(t *testing.T) {
	c := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, c)
	producer.ExpectSendMessageAndSucceed()

	p := kafkaMetricsProducer{
		producer:   producer,
		marshaller: &otlpMetricsPbMarshaller{},
	}
	t.Cleanup(func() {
		require.NoError(t, p.Close(context.Background()))
	})
	dropped, err := p.metricsDataPusher(context.Background(), testdata.GenerateMetricsTwoMetrics())
	require.NoError(t, err)
	assert.Equal(t, 0, dropped)
}

func TestMetricsDataPusher_err(t *testing.T) {
	c := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, c)
	expErr := fmt.Errorf("failed to send")
	producer.ExpectSendMessageAndFail(expErr)

	p := kafkaMetricsProducer{
		producer:   producer,
		marshaller: &otlpMetricsPbMarshaller{},
		logger:     zap.NewNop(),
	}
	t.Cleanup(func() {
		require.NoError(t, p.Close(context.Background()))
	})
	md := testdata.GenerateMetricsTwoMetrics()
	dropped, err := p.metricsDataPusher(context.Background(), md)
	assert.EqualError(t, err, expErr.Error())
	assert.Equal(t, md.MetricCount(), dropped)
}

func TestMetricsDataPusher_marshal_error(t *testing.T) {
	expErr := fmt.Errorf("failed to marshall")
	p := kafkaMetricsProducer{
		marshaller: &metricsErrorMarshaller{err: expErr},
		logger:     zap.NewNop(),
	}
	md := testdata.GenerateMetricsTwoMetrics()
	dropped, err := p.metricsDataPusher(context.Background(), md)
	require.Error(t, err)
	assert.Contains(t, err.Error(), expErr.Error())
	assert.Equal(t, md.MetricCount(), dropped)
}

type tracesErrorMarshaller struct {
	err error
}

type metricsErrorMarshaller struct {
	err error
}

func (e metricsErrorMarshaller) Marshal(_ pdata.Metrics) ([]Message, error) {
	return nil, e.err
}

func (e metricsErrorMarshaller) Encoding() string {
	panic("implement me")
}

var _ TracesMarshaller = (*tracesErrorMarshaller)(nil)

func (e tracesErrorMarshaller) Marshal(_ pdata.Traces) ([]Message, error) {
	return nil, e.err
}

func (e tracesErrorMarshaller) Encoding() string {
	panic("implement me")
}
