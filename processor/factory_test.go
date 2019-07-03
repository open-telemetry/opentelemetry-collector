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

package processor

import (
	"testing"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/configv2/configerror"
	"github.com/open-telemetry/opentelemetry-service/configv2/configmodels"
	"github.com/open-telemetry/opentelemetry-service/consumer"
)

type TestFactory struct {
}

// Type gets the type of the Processor config created by this factory.
func (f *TestFactory) Type() string {
	return "exampleoption"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *TestFactory) CreateDefaultConfig() configmodels.Processor {
	return nil
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *TestFactory) CreateTraceProcessor(
	logger *zap.Logger,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor,
) (TraceProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsProcessor creates a metrics processor based on this config.
func (f *TestFactory) CreateMetricsProcessor(
	logger *zap.Logger,
	nextConsumer consumer.MetricsConsumer,
	cfg configmodels.Processor,
) (MetricsProcessor, error) {
	return nil, configerror.ErrDataTypeIsNotSupported
}

func TestRegisterProcessorFactory(t *testing.T) {
	f := TestFactory{}
	err := RegisterFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetFactory(f.Type()) {
		t.Fatalf("cannot find factory")
	}

	// Verify that attempt to register a factory with duplicate name panics
	paniced := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				paniced = true
			}
		}()

		err = RegisterFactory(&f)
	}()

	if !paniced {
		t.Fatalf("must panic on double registration")
	}
}
