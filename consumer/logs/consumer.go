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

package logs // import "go.opentelemetry.io/collector/consumer/logs"

import (
	"context"

	"go.opentelemetry.io/collector/consumer/internal"
	"go.opentelemetry.io/collector/model/pdata/logs"
)

// Consumer is an interface that receives logs.Logs, processes it
// as needed, and sends it to the next processing node if any or to the destination.
type Consumer interface {
	internal.BaseConsumer
	// Consume receives logs.Logs for consumption.
	Consume(context.Context, logs.Logs) error
}

// ConsumeFunc is a helper function that is similar to Consume.
type ConsumeFunc func(context.Context, logs.Logs) error

// Consume calls f(ctx, l).
func (f ConsumeFunc) Consume(ctx context.Context, l logs.Logs) error {
	return f(ctx, l)
}

type base struct {
	*internal.BaseImpl
	ConsumeFunc
}

// NewConsumer returns a Logs configured with the provided options.
func NewConsumer(consume ConsumeFunc, options ...internal.Option) (Consumer, error) {
	if consume == nil {
		return nil, internal.ErrNilFunc
	}
	return &base{
		BaseImpl:    internal.NewBaseImpl(options...),
		ConsumeFunc: consume,
	}, nil
}
