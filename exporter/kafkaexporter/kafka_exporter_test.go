// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"time"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

const exporterName = "test"

func TestNewExporter_wrong_version(t *testing.T) {
	c := Config{ProtocolVersion: "0.0.0"}
	exp, err := newExporter(c, component.ExporterCreateParams{})
	assert.Error(t, err)
	assert.Nil(t, exp)
}

func TestTraceDataPusher(t *testing.T) {
	views := MetricViews()
	view.Register(views...)
	defer view.Unregister(views...)
	c := sarama.NewConfig()
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	producer := mocks.NewAsyncProducer(t, c)
	producer.ExpectInputAndSucceed()

	p := kafkaProducer{
		asyncProducer: producer,
		marshaller:    &protoMarshaller{},
	}
	p.processSendResults(exporterName)
	defer p.Close(context.Background())
	droppedSpans, err := p.traceDataPusher(context.Background(), testdata.GenerateTraceDataTwoSpansSameResource())
	require.NoError(t, err)
	assert.Equal(t, 0, droppedSpans)

	// wait for success and error goroutines to finish
	time.Sleep(time.Millisecond * 500)
	viewData, err := view.RetrieveData(statSendSuccess.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)
}

func TestTraceDataPusher_err(t *testing.T) {
	views := MetricViews()
	view.Register(views...)
	defer view.Unregister(views...)
	c := sarama.NewConfig()
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	producer := mocks.NewAsyncProducer(t, c)
	producer.ExpectInputAndFail(fmt.Errorf("failed to send"))

	p := kafkaProducer{
		asyncProducer: producer,
		marshaller:    &protoMarshaller{},
		logger:        zap.NewNop(),
	}
	p.processSendResults(exporterName)
	defer p.Close(context.Background())
	droppedSpans, err := p.traceDataPusher(context.Background(), testdata.GenerateTraceDataTwoSpansSameResource())
	require.NoError(t, err)
	assert.Equal(t, 0, droppedSpans)

	// wait for success and error goroutines to finish
	time.Sleep(time.Millisecond * 500)
	viewData, err := view.RetrieveData(statSendErr.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)
}

func TestTraceDataPusher_remove(t *testing.T) {
	c := sarama.NewConfig()
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	c.Net.Proxy.Enable = true
	producer := mocks.NewAsyncProducer(t, c)
	producer.ExpectInputAndFail(fmt.Errorf("failed to send"))
}
