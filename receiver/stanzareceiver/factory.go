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

	stanza "github.com/observiq/stanza/agent"
	"github.com/observiq/stanza/entry"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	typeStr = "stanza"
)

// NewFactory creates a factory for Stanza receiver.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithLogs(createLogsReceiver))
}

func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: configmodels.Type(typeStr),
			NameVal: typeStr,
		},
	}
}

// CreateLogsReceiver creates a logs receiver based on provided config
func createLogsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogsConsumer,
) (component.LogsReceiver, error) {

	obsConfig := cfg.(*Config)
	logsChan := make(chan *entry.Entry)
	buildParams := map[string]interface{}{logsChannelID: logsChan}
	logAgent, err := stanza.NewLogAgent(
		&stanza.Config{Pipeline: obsConfig.Pipeline},
		params.Logger.Sugar(),
		obsConfig.PluginsDir,
		obsConfig.OffsetsFile,
		buildParams)
	if err != nil {
		return nil, err
	}

	return &stanzareceiver{
		agent:    logAgent,
		logsChan: logsChan,
		consumer: nextConsumer,
		logger:   params.Logger,
		done:     make(chan struct{}),
	}, nil
}
