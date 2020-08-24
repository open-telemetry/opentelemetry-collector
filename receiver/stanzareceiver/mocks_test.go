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
	"github.com/observiq/stanza/pipeline"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// This file implements some useful testing components
func init() {
	operator.Register("unstartable_operator", func() operator.Builder { return NewUnstartableConfig() })
}

// UnstartableConfig is the configuration of an unstartable mock operator
type UnstartableConfig struct {
	helper.OutputConfig `yaml:",inline"`
}

// UnstartableOperator is an operator that will build but not start
// While this is not expected behavior, it is possible that build-time
// validation could be invalidated before Start() is called
type UnstartableOperator struct {
	helper.OutputOperator
}

func newUnstartableParams() pipeline.Params {
	return pipeline.Params(map[string]interface{}{"type": "unstartable_operator"})
}

// NewUnstartableConfig creates new output config
func NewUnstartableConfig() *UnstartableConfig {
	return &UnstartableConfig{
		OutputConfig: helper.NewOutputConfig("unstartable_operator", "unstartable_operator"),
	}
}

// Build will build an unstartable operator
func (c *UnstartableConfig) Build(context operator.BuildContext) (operator.Operator, error) {
	o, _ := c.OutputConfig.Build(context)
	return &UnstartableOperator{OutputOperator: o}, nil
}

// Start will return an error
func (o *UnstartableOperator) Start() error {
	return fmt.Errorf("something very unusual happened")
}

// Process will return nil
func (o *UnstartableOperator) Process(ctx context.Context, entry *entry.Entry) error {
	return nil
}

type mockLogsConsumer struct {
	received int
}

func (m *mockLogsConsumer) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	m.received++
	return nil
}

type mockLogsRejecter struct {
	rejected int
}

func (m *mockLogsRejecter) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	m.rejected++
	return fmt.Errorf("no")
}
