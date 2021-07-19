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

package kafkareceiver

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/obsreport"
)

func TestNewTracesReceiver_version_err(t *testing.T) {
	c := Config{
		Encoding:        defaultEncoding,
		ProtocolVersion: "none",
	}
	r, err := newTracesReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultTracesUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestNewTracesReceiver_encoding_err(t *testing.T) {
	c := Config{
		Encoding: "foo",
	}
	r, err := newTracesReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultTracesUnmarshalers(), consumertest.NewNop())
	require.Error(t, err)
	assert.Nil(t, r)
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
}

func TestNewTracesReceiver_err_auth_type(t *testing.T) {
	c := Config{
		ProtocolVersion: "2.0.0",
		Authentication: kafkaexporter.Authentication{
			TLS: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: "/doesnotexist",
				},
			},
		},
		Encoding: defaultEncoding,
		Metadata: kafkaexporter.Metadata{
			Full: false,
		},
	}
	r, err := newTracesReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultTracesUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load TLS config")
	assert.Nil(t, r)
}

func TestTracesReceiverStart(t *testing.T) {
	c := kafkaTracesConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}

	require.NoError(t, c.Start(context.Background(), nil))
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestTracesReceiverStartConsume(t *testing.T) {
	c := kafkaTracesConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancelFunc
	require.NoError(t, c.Shutdown(context.Background()))
	err := c.consumeLoop(ctx, &tracesConsumerGroupHandler{
		ready: make(chan bool),
	})
	assert.EqualError(t, err, context.Canceled.Error())
}

func TestTracesReceiver_error(t *testing.T) {
	zcore, logObserver := observer.New(zapcore.ErrorLevel)
	logger := zap.New(zcore)

	expectedErr := errors.New("handler error")
	c := kafkaTracesConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        logger,
		consumerGroup: &testConsumerGroup{err: expectedErr},
	}

	require.NoError(t, c.Start(context.Background(), nil))
	require.NoError(t, c.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return logObserver.FilterField(zap.Error(expectedErr)).Len() > 0
	}, 10*time.Second, time.Millisecond*100)
}

func TestTracesConsumerGroupHandler(t *testing.T) {
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	c := tracesConsumerGroupHandler{
		unmarshaler:  newPdataTracesUnmarshaler(otlp.NewProtobufTracesUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	testSession := testConsumerGroupSession{}
	require.NoError(t, c.Setup(testSession))
	_, ok := <-c.ready
	assert.False(t, ok)
	viewData, err := view.RetrieveData(statPartitionStart.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	require.NoError(t, c.Cleanup(testSession))
	viewData, err = view.RetrieveData(statPartitionClose.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	groupClaim := testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		require.NoError(t, c.ConsumeClaim(testSession, groupClaim))
		wg.Done()
	}()

	groupClaim.messageChan <- &sarama.ConsumerMessage{}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestTracesConsumerGroupHandler_error_unmarshal(t *testing.T) {
	c := tracesConsumerGroupHandler{
		unmarshaler:  newPdataTracesUnmarshaler(otlp.NewProtobufTracesUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		err := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		require.Error(t, err)
		wg.Done()
	}()
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: []byte("!@#")}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestTracesConsumerGroupHandler_error_nextConsumer(t *testing.T) {
	consumerError := errors.New("failed to consume")
	c := tracesConsumerGroupHandler{
		unmarshaler:  newPdataTracesUnmarshaler(otlp.NewProtobufTracesUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewErr(consumerError),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		e := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		assert.EqualError(t, e, consumerError.Error())
		wg.Done()
	}()

	td := pdata.NewTraces()
	td.ResourceSpans().AppendEmpty()
	bts, err := otlp.NewProtobufTracesMarshaler().MarshalTraces(td)
	require.NoError(t, err)
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: bts}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestNewMetricsReceiver_version_err(t *testing.T) {
	c := Config{
		Encoding:        defaultEncoding,
		ProtocolVersion: "none",
	}
	r, err := newMetricsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultMetricsUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestNewMetricsReceiver_encoding_err(t *testing.T) {
	c := Config{
		Encoding: "foo",
	}
	r, err := newMetricsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultMetricsUnmarshalers(), consumertest.NewNop())
	require.Error(t, err)
	assert.Nil(t, r)
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
}

func TestNewMetricsExporter_err_auth_type(t *testing.T) {
	c := Config{
		ProtocolVersion: "2.0.0",
		Authentication: kafkaexporter.Authentication{
			TLS: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: "/doesnotexist",
				},
			},
		},
		Encoding: defaultEncoding,
		Metadata: kafkaexporter.Metadata{
			Full: false,
		},
	}
	r, err := newMetricsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultMetricsUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load TLS config")
	assert.Nil(t, r)
}

func TestMetricsReceiverStart(t *testing.T) {
	c := kafkaMetricsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}

	require.NoError(t, c.Start(context.Background(), nil))
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestMetricsReceiverStartConsume(t *testing.T) {
	c := kafkaMetricsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancelFunc
	require.NoError(t, c.Shutdown(context.Background()))
	err := c.consumeLoop(ctx, &logsConsumerGroupHandler{
		ready: make(chan bool),
	})
	assert.EqualError(t, err, context.Canceled.Error())
}

func TestMetricsReceiver_error(t *testing.T) {
	zcore, logObserver := observer.New(zapcore.ErrorLevel)
	logger := zap.New(zcore)

	expectedErr := errors.New("handler error")
	c := kafkaMetricsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        logger,
		consumerGroup: &testConsumerGroup{err: expectedErr},
	}

	require.NoError(t, c.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, c.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return logObserver.FilterField(zap.Error(expectedErr)).Len() > 0
	}, 10*time.Second, time.Millisecond*100)
}

func TestMetricsConsumerGroupHandler(t *testing.T) {
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	c := metricsConsumerGroupHandler{
		unmarshaler:  newPdataMetricsUnmarshaler(otlp.NewProtobufMetricsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	testSession := testConsumerGroupSession{}
	require.NoError(t, c.Setup(testSession))
	_, ok := <-c.ready
	assert.False(t, ok)
	viewData, err := view.RetrieveData(statPartitionStart.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	require.NoError(t, c.Cleanup(testSession))
	viewData, err = view.RetrieveData(statPartitionClose.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	groupClaim := testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		require.NoError(t, c.ConsumeClaim(testSession, groupClaim))
		wg.Done()
	}()

	groupClaim.messageChan <- &sarama.ConsumerMessage{}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestMetricsConsumerGroupHandler_error_unmarshal(t *testing.T) {
	c := metricsConsumerGroupHandler{
		unmarshaler:  newPdataMetricsUnmarshaler(otlp.NewProtobufMetricsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		err := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		require.Error(t, err)
		wg.Done()
	}()
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: []byte("!@#")}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestMetricsConsumerGroupHandler_error_nextConsumer(t *testing.T) {
	consumerError := errors.New("failed to consume")
	c := metricsConsumerGroupHandler{
		unmarshaler:  newPdataMetricsUnmarshaler(otlp.NewProtobufMetricsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewErr(consumerError),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		e := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		assert.EqualError(t, e, consumerError.Error())
		wg.Done()
	}()

	ld := testdata.GenerateMetricsOneMetric()
	bts, err := otlp.NewProtobufMetricsMarshaler().MarshalMetrics(ld)
	require.NoError(t, err)
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: bts}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestNewLogsReceiver_version_err(t *testing.T) {
	c := Config{
		Encoding:        defaultEncoding,
		ProtocolVersion: "none",
	}
	r, err := newLogsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultLogsUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Nil(t, r)
}

func TestNewLogsReceiver_encoding_err(t *testing.T) {
	c := Config{
		Encoding: "foo",
	}
	r, err := newLogsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultLogsUnmarshalers(), consumertest.NewNop())
	require.Error(t, err)
	assert.Nil(t, r)
	assert.EqualError(t, err, errUnrecognizedEncoding.Error())
}

func TestNewLogsExporter_err_auth_type(t *testing.T) {
	c := Config{
		ProtocolVersion: "2.0.0",
		Authentication: kafkaexporter.Authentication{
			TLS: &configtls.TLSClientSetting{
				TLSSetting: configtls.TLSSetting{
					CAFile: "/doesnotexist",
				},
			},
		},
		Encoding: defaultEncoding,
		Metadata: kafkaexporter.Metadata{
			Full: false,
		},
	}
	r, err := newLogsReceiver(c, componenttest.NewNopReceiverCreateSettings(), defaultLogsUnmarshalers(), consumertest.NewNop())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load TLS config")
	assert.Nil(t, r)
}

func TestLogsReceiverStart(t *testing.T) {
	c := kafkaLogsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}

	require.NoError(t, c.Start(context.Background(), nil))
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestLogsReceiverStartConsume(t *testing.T) {
	c := kafkaLogsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        zap.NewNop(),
		consumerGroup: &testConsumerGroup{},
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancelFunc
	require.NoError(t, c.Shutdown(context.Background()))
	err := c.consumeLoop(ctx, &logsConsumerGroupHandler{
		ready: make(chan bool),
	})
	assert.EqualError(t, err, context.Canceled.Error())
}

func TestLogsReceiver_error(t *testing.T) {
	zcore, logObserver := observer.New(zapcore.ErrorLevel)
	logger := zap.New(zcore)

	expectedErr := errors.New("handler error")
	c := kafkaLogsConsumer{
		nextConsumer:  consumertest.NewNop(),
		logger:        logger,
		consumerGroup: &testConsumerGroup{err: expectedErr},
	}

	require.NoError(t, c.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, c.Shutdown(context.Background()))
	assert.Eventually(t, func() bool {
		return logObserver.FilterField(zap.Error(expectedErr)).Len() > 0
	}, 10*time.Second, time.Millisecond*100)
}

func TestLogsConsumerGroupHandler(t *testing.T) {
	views := MetricViews()
	require.NoError(t, view.Register(views...))
	defer view.Unregister(views...)

	c := logsConsumerGroupHandler{
		unmarshaler:  newPdataLogsUnmarshaler(otlp.NewProtobufLogsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	testSession := testConsumerGroupSession{}
	require.NoError(t, c.Setup(testSession))
	_, ok := <-c.ready
	assert.False(t, ok)
	viewData, err := view.RetrieveData(statPartitionStart.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData := viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	require.NoError(t, c.Cleanup(testSession))
	viewData, err = view.RetrieveData(statPartitionClose.Name())
	require.NoError(t, err)
	assert.Equal(t, 1, len(viewData))
	distData = viewData[0].Data.(*view.SumData)
	assert.Equal(t, float64(1), distData.Value)

	groupClaim := testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		require.NoError(t, c.ConsumeClaim(testSession, groupClaim))
		wg.Done()
	}()

	groupClaim.messageChan <- &sarama.ConsumerMessage{}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestLogsConsumerGroupHandler_error_unmarshal(t *testing.T) {
	c := logsConsumerGroupHandler{
		unmarshaler:  newPdataLogsUnmarshaler(otlp.NewProtobufLogsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewNop(),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		err := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		require.Error(t, err)
		wg.Done()
	}()
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: []byte("!@#")}
	close(groupClaim.messageChan)
	wg.Wait()
}

func TestLogsConsumerGroupHandler_error_nextConsumer(t *testing.T) {
	consumerError := errors.New("failed to consume")
	c := logsConsumerGroupHandler{
		unmarshaler:  newPdataLogsUnmarshaler(otlp.NewProtobufLogsUnmarshaler(), defaultEncoding),
		logger:       zap.NewNop(),
		ready:        make(chan bool),
		nextConsumer: consumertest.NewErr(consumerError),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{}),
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	groupClaim := &testConsumerGroupClaim{
		messageChan: make(chan *sarama.ConsumerMessage),
	}
	go func() {
		e := c.ConsumeClaim(testConsumerGroupSession{}, groupClaim)
		assert.EqualError(t, e, consumerError.Error())
		wg.Done()
	}()

	ld := testdata.GenerateLogsOneLogRecord()
	bts, err := otlp.NewProtobufLogsMarshaler().MarshalLogs(ld)
	require.NoError(t, err)
	groupClaim.messageChan <- &sarama.ConsumerMessage{Value: bts}
	close(groupClaim.messageChan)
	wg.Wait()
}

type testConsumerGroupClaim struct {
	messageChan chan *sarama.ConsumerMessage
}

var _ sarama.ConsumerGroupClaim = (*testConsumerGroupClaim)(nil)

const (
	testTopic               = "otlp_spans"
	testPartition           = 5
	testInitialOffset       = 6
	testHighWatermarkOffset = 4
)

func (t testConsumerGroupClaim) Topic() string {
	return testTopic
}

func (t testConsumerGroupClaim) Partition() int32 {
	return testPartition
}

func (t testConsumerGroupClaim) InitialOffset() int64 {
	return testInitialOffset
}

func (t testConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return testHighWatermarkOffset
}

func (t testConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return t.messageChan
}

type testConsumerGroupSession struct {
}

func (t testConsumerGroupSession) Commit() {
	panic("implement me")
}

var _ sarama.ConsumerGroupSession = (*testConsumerGroupSession)(nil)

func (t testConsumerGroupSession) Claims() map[string][]int32 {
	panic("implement me")
}

func (t testConsumerGroupSession) MemberID() string {
	panic("implement me")
}

func (t testConsumerGroupSession) GenerationID() int32 {
	panic("implement me")
}

func (t testConsumerGroupSession) MarkOffset(string, int32, int64, string) {
	panic("implement me")
}

func (t testConsumerGroupSession) ResetOffset(string, int32, int64, string) {
	panic("implement me")
}

func (t testConsumerGroupSession) MarkMessage(*sarama.ConsumerMessage, string) {}

func (t testConsumerGroupSession) Context() context.Context {
	return context.Background()
}

type testConsumerGroup struct {
	once sync.Once
	err  error
}

var _ sarama.ConsumerGroup = (*testConsumerGroup)(nil)

func (t *testConsumerGroup) Consume(_ context.Context, _ []string, handler sarama.ConsumerGroupHandler) error {
	t.once.Do(func() {
		handler.Setup(testConsumerGroupSession{})
	})
	return t.err
}

func (t *testConsumerGroup) Errors() <-chan error {
	panic("implement me")
}

func (t *testConsumerGroup) Close() error {
	return nil
}
