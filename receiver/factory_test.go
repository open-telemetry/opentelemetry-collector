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

package receiver

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/configv2/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
)

type TestFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *TestFactory) Type() string {
	return "examplereceiver"
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *TestFactory) CustomUnmarshaler() CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *TestFactory) CreateDefaultConfig() configmodels.Receiver {
	return nil
}

// CreateTraceReceiver creates a trace receiver based on this config.
func (f *TestFactory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (TraceReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

// CreateMetricsReceiver creates a metrics receiver based on this config.
func (f *TestFactory) CreateMetricsReceiver(
	logger *zap.Logger,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (MetricsReceiver, error) {
	// Not used for this test, just return nil
	return nil, nil
}

func TestRegisterFactory(t *testing.T) {
	f := TestFactory{}
	err := RegisterFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetFactory(f.Type()) {
		t.Fatalf("cannot find factory")
	}

	// Verify that attempt to register a factory with duplicate name panics
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()

		err = RegisterFactory(&f)
	}()

	if !panicked {
		t.Fatalf("must panic on double registration")
	}
}
