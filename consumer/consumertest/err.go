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

package consumertest

import (
	"context"

	"go.opentelemetry.io/collector/model/pdata"
)

type errConsumer struct {
	nonMutatingConsumer
	err error
}

func (er *errConsumer) unexported() {}

func (er *errConsumer) ConsumeTraces(context.Context, pdata.Traces) error {
	return er.err
}

func (er *errConsumer) ConsumeMetrics(context.Context, pdata.Metrics) error {
	return er.err
}

func (er *errConsumer) ConsumeLogs(context.Context, pdata.Logs) error {
	return er.err
}

// NewErr returns a Consumer that just drops all received data and returns the specified error to Consume* callers.
func NewErr(err error) Consumer {
	return &errConsumer{err: err}
}
