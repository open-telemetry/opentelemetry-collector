// Copyright 2021 OpenTelemetry Authors
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

package syslogreceiver

import (
	"context"
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/stanza"
)

func TestSyslogWithTcp(t *testing.T) {
	testSyslog(t, testdataConfigYamlAsMap())
}

func TestSyslogWithUdp(t *testing.T) {
	testSyslog(t, testdataUDPConfig())
}

func testSyslog(t *testing.T, cfg *SysLogConfig) {
	numLogs := 5

	f := NewFactory()
	params := component.ReceiverCreateParams{Logger: zaptest.NewLogger(t)}
	sink := new(consumertest.LogsSink)
	rcvr, err := f.CreateLogsReceiver(context.Background(), params, cfg, sink)
	require.NoError(t, err)
	require.NoError(t, rcvr.Start(context.Background(), componenttest.NewNopHost()))

	var conn net.Conn
	if cfg.Input["tcp"] != nil {
		conn, err = net.Dial("tcp", "0.0.0.0:29018")
		require.NoError(t, err)
	} else {
		conn, err = net.Dial("udp", "0.0.0.0:29018")
		require.NoError(t, err)
	}

	for i := 0; i < numLogs; i++ {
		msg := fmt.Sprintf("<86>1 2021-02-28T00:0%d:02.003Z 192.168.1.1 SecureAuth0 23108 ID52020 [SecureAuth@27389] test msg %d\n", i, i)
		_, err = conn.Write([]byte(msg))
		require.NoError(t, err)
	}
	require.NoError(t, conn.Close())

	require.Eventually(t, expectNLogs(sink, numLogs), 2*time.Second, time.Millisecond)
	require.NoError(t, rcvr.Shutdown(context.Background()))
	require.Len(t, sink.AllLogs(), 1)

	resourceLogs := sink.AllLogs()[0].ResourceLogs().At(0)
	logs := resourceLogs.InstrumentationLibraryLogs().At(0).Logs()

	for i := 0; i < numLogs; i++ {
		log := logs.At(i)

		require.Equal(t, log.Timestamp(), pdata.Timestamp(1614470402003000000+i*60*1000*1000*1000))
		msg, ok := log.Body().MapVal().Get("message")
		require.True(t, ok)
		require.Equal(t, msg.StringVal(), fmt.Sprintf("test msg %d", i))
	}
}

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.Nil(t, err)

	factory := NewFactory()
	factories.Receivers[config.Type(typeStr)] = factory
	cfg, err := configtest.LoadConfigFile(
		t, path.Join(".", "testdata", "config.yaml"), factories,
	)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 1)
	assert.Equal(t, testdataConfigYamlAsMap(), cfg.Receivers["syslog"])
}

func testdataConfigYamlAsMap() *SysLogConfig {
	return &SysLogConfig{
		BaseConfig: stanza.BaseConfig{
			ReceiverSettings: config.ReceiverSettings{
				TypeVal: "syslog",
				NameVal: "syslog",
			},
			Operators: stanza.OperatorConfigs{},
			Converter: stanza.ConverterConfig{
				FlushInterval: 100 * time.Millisecond,
				WorkerCount:   1,
			},
		},
		Input: stanza.InputConfig{
			"tcp": map[string]interface{}{
				"listen_address": "0.0.0.0:29018",
			},
			"protocol": "rfc5424",
		},
	}
}

func testdataUDPConfig() *SysLogConfig {
	return &SysLogConfig{
		BaseConfig: stanza.BaseConfig{
			ReceiverSettings: config.ReceiverSettings{
				TypeVal: "syslog",
				NameVal: "syslog",
			},
			Operators: stanza.OperatorConfigs{},
			Converter: stanza.ConverterConfig{
				FlushInterval: 100 * time.Millisecond,
				WorkerCount:   1,
			},
		},
		Input: stanza.InputConfig{
			"udp": map[string]interface{}{
				"listen_address": "0.0.0.0:29018",
			},
			"protocol": "rfc5424",
		},
	}
}

func TestDecodeInputConfigFailure(t *testing.T) {
	params := component.ReceiverCreateParams{
		Logger: zap.NewNop(),
	}
	sink := new(consumertest.LogsSink)
	factory := NewFactory()
	badCfg := &SysLogConfig{
		BaseConfig: stanza.BaseConfig{
			ReceiverSettings: config.ReceiverSettings{
				TypeVal: "syslog",
				NameVal: "syslog",
			},
			Operators: stanza.OperatorConfigs{},
		},
		Input: stanza.InputConfig{
			"tcp": map[string]interface{}{
				"max_buffer_size": "0.1.0.1-",
			},
			"protocol": "rfc5424",
		},
	}
	receiver, err := factory.CreateLogsReceiver(context.Background(), params, badCfg, sink)
	require.Error(t, err, "receiver creation should fail if input config isn't valid")
	require.Nil(t, receiver, "receiver creation should fail if input config isn't valid")
}

func expectNLogs(sink *consumertest.LogsSink, expected int) func() bool {
	return func() bool {
		return sink.LogRecordsCount() == expected
	}
}
