package stanzareceiver

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

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

const logsChannelID = "logs_channel"

// This file implements an operator that consumes logs from the observiq pipeline
func init() {
	operator.Register("receiver_output", defaultCfg)
}

func defaultCfg() operator.Builder {
	return NewReceiverOutputConfig("receiver_output")
}

// NewReceiverOutputConfig creates new output config
func NewReceiverOutputConfig(operatorID string) *ReceiverOutputConfig {
	return &ReceiverOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "receiver_output"),
	}
}

// ReceiverOutputConfig is the configuration of an receiver output operator
type ReceiverOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
}

// Build will build an receiver output operator
func (c ReceiverOutputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if context.Parameters == nil {
		return nil, fmt.Errorf("Build parameters not found in build context")
	}

	buildParam, ok := context.Parameters[logsChannelID]
	if !ok {
		return nil, fmt.Errorf("%s not found in build context", logsChannelID)
	}

	outChan, ok := buildParam.(chan *entry.Entry)
	if !ok {
		return nil, fmt.Errorf("%s not of correct type", logsChannelID)
	}

	receiverOutput := &ReceiverOutput{
		OutputOperator: outputOperator,
		outChan:        outChan,
	}

	return receiverOutput, nil
}

// ReceiverOutput is an operator that sends entries to channel
type ReceiverOutput struct {
	helper.OutputOperator
	outChan chan *entry.Entry
}

// Process will write an entry to the output channel
func (c *ReceiverOutput) Process(ctx context.Context, entry *entry.Entry) error {
	c.outChan <- entry
	return nil
}
