// Copyright 2019, OpenCensus Authors
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

package factories

import (
	"testing"

	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
)

type ExampleReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ExampleReceiverFactory) Type() string {
	return "examplereceiver"
}

// CreateDefaultConfig creates the default configuration for the Receiver.
func (f *ExampleReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return nil
}

func TestRegisterReceiverFactory(t *testing.T) {
	f := ExampleReceiverFactory{}
	err := RegisterReceiverFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetReceiverFactory(f.Type()) {
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

		err = RegisterReceiverFactory(&f)
	}()

	if !panicked {
		t.Fatalf("must panic on double registration")
	}
}

type ExampleExporterFactory struct {
}

// Type gets the type of the Exporter config created by this factory.
func (f *ExampleExporterFactory) Type() string {
	return "exampleexporter"
}

// CreateDefaultConfig creates the default configuration for the Exporter.
func (f *ExampleExporterFactory) CreateDefaultConfig() configmodels.Exporter {
	return nil
}

func TestRegisterExporterFactory(t *testing.T) {
	f := ExampleExporterFactory{}
	err := RegisterExporterFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetExporterFactory(f.Type()) {
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

		err = RegisterExporterFactory(&f)
	}()

	if !paniced {
		t.Fatalf("must panic on double registration")
	}
}

type ExampleOptionFactory struct {
}

// Type gets the type of the Option config created by this factory.
func (f *ExampleOptionFactory) Type() string {
	return "exampleoption"
}

// CreateDefaultConfig creates the default configuration for the Processor.
func (f *ExampleOptionFactory) CreateDefaultConfig() configmodels.Processor {
	return nil
}

func TestRegisterOptionFactory(t *testing.T) {
	f := ExampleOptionFactory{}
	err := RegisterProcessorFactory(&f)
	if err != nil {
		t.Fatalf("cannot register factory")
	}

	if &f != GetProcessorFactory(f.Type()) {
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

		err = RegisterProcessorFactory(&f)
	}()

	if !paniced {
		t.Fatalf("must panic on double registration")
	}
}
