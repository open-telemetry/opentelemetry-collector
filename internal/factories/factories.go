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
	"fmt"

	"github.com/census-instrumentation/opencensus-service/internal/configmodels"
)

///////////////////////////////////////////////////////////////////////////////
// Receiver factory and its registry.

// ReceiverFactory is factory interface for receivers. Note: only configuration-related
// functionality exists for now. We will add more factory functionality in the future.
type ReceiverFactory interface {
	// Type gets the type of the Receiver created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Receiver.
	CreateDefaultConfig() configmodels.Receiver
}

// List of registered receiver factories.
var receiverFactories = make(map[string]ReceiverFactory)

// RegisterReceiverFactory registers a receiver factory.
func RegisterReceiverFactory(factory ReceiverFactory) error {
	if receiverFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate receiver factory %q", factory.Type()))
	}

	receiverFactories[factory.Type()] = factory
	return nil
}

// GetReceiverFactory gets a receiver factory by type string.
func GetReceiverFactory(typeStr string) ReceiverFactory {
	return receiverFactories[typeStr]
}

///////////////////////////////////////////////////////////////////////////////
// Exporter factory and its registry.

// ExporterFactory is factory interface for exporters. Note: only configuration-related
// functionality exists for now. We will add more factory functionality in the future.
type ExporterFactory interface {
	// Type gets the type of the Exporter created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Exporter.
	CreateDefaultConfig() configmodels.Exporter
}

// List of registered exporter factories.
var exporterFactories = make(map[string]ExporterFactory)

// RegisterExporterFactory registers a exporter factory.
func RegisterExporterFactory(factory ExporterFactory) error {
	if exporterFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate exporter factory %q", factory.Type()))
	}

	exporterFactories[factory.Type()] = factory
	return nil
}

// GetExporterFactory gets a exporter factory by type string.
func GetExporterFactory(typeStr string) ExporterFactory {
	return exporterFactories[typeStr]
}

///////////////////////////////////////////////////////////////////////////////
// Option factory and its registry.

// OptionFactory is factory interface for options. Note: only configuration-related
// functionality exists for now. We will add more factory functionality in the future.
type OptionFactory interface {
	// Type gets the type of the Option created by this factory.
	Type() string

	// CreateDefaultConfig creates the default configuration for the Option.
	CreateDefaultConfig() configmodels.Processor
}

// List of registered option factories.
var optionFactories = make(map[string]OptionFactory)

// RegisterProcessorFactory registers a option factory.
func RegisterProcessorFactory(factory OptionFactory) error {
	if optionFactories[factory.Type()] != nil {
		panic(fmt.Sprintf("duplicate option factory %q", factory.Type()))
	}

	optionFactories[factory.Type()] = factory
	return nil
}

// GetProcessorFactory gets a option factory by type string.
func GetProcessorFactory(typeStr string) OptionFactory {
	return optionFactories[typeStr]
}
