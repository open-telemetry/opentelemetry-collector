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

package stanzareceiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/pipeline"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v2"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestStart(t *testing.T) {
	params := component.ReceiverCreateParams{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}
	receiver, _ := createLogsReceiver(context.Background(), params, createDefaultConfig(), &mockConsumer)

	err := receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "receiver start failed")

	obsReceiver := receiver.(*stanzareceiver)
	obsReceiver.emitter.LogChan() <- entry.New()
	receiver.Shutdown(context.Background())
	require.Equal(t, 1, mockConsumer.received, "one log entry expected")
}

func TestHandleStartError(t *testing.T) {
	params := component.ReceiverCreateParams{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	cfg := createDefaultConfig().(*Config)
	cfg.Pipeline = append(cfg.Pipeline, newUnstartableParams())

	receiver, err := createLogsReceiver(context.Background(), params, cfg, &mockConsumer)
	require.NoError(t, err, "receiver should successfully build")

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err, "receiver fails to start under rare circumstances")
}

func TestHandleConsumeError(t *testing.T) {
	params := component.ReceiverCreateParams{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsRejecter{}
	receiver, _ := createLogsReceiver(context.Background(), params, createDefaultConfig(), &mockConsumer)

	err := receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "receiver start failed")

	obsReceiver := receiver.(*stanzareceiver)
	obsReceiver.emitter.LogChan() <- entry.New()
	receiver.Shutdown(context.Background())
	require.Equal(t, 1, mockConsumer.rejected, "one log entry expected")
}

func BenchmarkPipelineSimple(b *testing.B) {

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		b.Errorf(err.Error())
		b.FailNow()
	}

	filePath := filepath.Join(tempDir, "bench.log")

	pipelineYaml := fmt.Sprintf(`
- type: file_input
  include:
    - %s
  start_at: beginning`,
		filePath)

	pipelineCfg := pipeline.Config{}
	require.NoError(b, yaml.Unmarshal([]byte(pipelineYaml), &pipelineCfg))

	emitter := NewLogEmitter(zap.NewNop().Sugar())
	defer emitter.Stop()

	buildContext := testutil.NewBuildContext(b)
	buildContext.Logger = zap.NewNop().Sugar() // be quiet

	pl, err := pipelineCfg.BuildPipeline(buildContext, emitter)
	require.NoError(b, err)

	// Populate the file that will be consumed
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		file.WriteString("testlog\n")
	}

	// // Run the actual benchmark
	b.ResetTimer()
	require.NoError(b, pl.Start())
	for i := 0; i < b.N; i++ {
		<-emitter.LogChan()
	}
}

func BenchmarkPipelineComplex(b *testing.B) {

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		b.Errorf(err.Error())
		b.FailNow()
	}

	filePath := filepath.Join(tempDir, "bench.log")

	fileInputYaml := fmt.Sprintf(`
- type: file_input
  include:
    - %s
  start_at: beginning`, filePath)

	regexParserYaml := `
- type: regex_parser
  regex: '(?P<remote_host>[^\s]+) - (?P<remote_user>[^\s]+) \[(?P<timestamp>[^\]]+)\] "(?P<http_method>[A-Z]+) (?P<path>[^\s]+)[^"]+" (?P<http_status>\d+) (?P<bytes_sent>[^\s]+)'
  timestamp:
    parse_from: timestamp
    layout: '%d/%b/%Y:%H:%M:%S %z'
  severity:
    parse_from: http_status
    preserve: true
    mapping:
      critical: 5xx
      error: 4xx
      info: 3xx
      debug: 2xx`

	pipelineYaml := fmt.Sprintf("%s%s", fileInputYaml, regexParserYaml)

	pipelineCfg := pipeline.Config{}
	require.NoError(b, yaml.Unmarshal([]byte(pipelineYaml), &pipelineCfg))

	emitter := NewLogEmitter(zap.NewNop().Sugar())
	defer emitter.Stop()

	buildContext := testutil.NewBuildContext(b)
	buildContext.Logger = zap.NewNop().Sugar() // be quiet

	pl, err := pipelineCfg.BuildPipeline(buildContext, emitter)
	require.NoError(b, err)

	// Populate the file that will be consumed
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		file.WriteString("10.33.121.119 - - [11/Aug/2020:00:00:00 -0400] \"GET /index.html HTTP/1.1\" 404 761\n")
	}

	// // Run the actual benchmark
	b.ResetTimer()
	require.NoError(b, pl.Start())
	for i := 0; i < b.N; i++ {
		<-emitter.LogChan()
	}
}
