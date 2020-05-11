package hwpipeline

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"go.uber.org/zap"
)

// Consumer ...
type Consumer interface {
	ConsumeHW(HelloWorld) error
}

// Processor ...
type Processor interface {
	component.Processor
	Consumer
}

// Exporter ...
type Exporter interface {
	component.Exporter
	Consumer
}

// ReceiverFactory ...
type ReceiverFactory interface {
	CreateHWReceiver(Consumer) component.Receiver
}

// ProcessorFactory ...
type ProcessorFactory interface {
	CreateHWProcessor(Consumer) Processor
}

// ExporterFactory ...
type ExporterFactory interface {
	CreateHWExporter() Exporter
}

// Factory ...
type Factory struct{}

// Type ...
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type("helloworld")
}

// CreateReceiver ...
func (f *Factory) CreateReceiver(
	ctx context.Context,
	factoryBase component.ReceiverFactoryBase,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer component.Consumer,
) (component.Receiver, error) {
	if factory, ok := factoryBase.(ReceiverFactory); ok {
		if nextConsumer, ok := nextConsumer.(Consumer); ok {
			return factory.CreateHWReceiver(nextConsumer), nil
		}
	}
	return nil, fmt.Errorf("%v is not a valid hello world receiver factory", factoryBase)
}

// CreateExporter ...
func (f *Factory) CreateExporter(
	ctx context.Context,
	factoryBase component.ExporterFactoryBase,
	logger *zap.Logger,
	config configmodels.Exporter,
) (component.Exporter, error) {
	if factory, ok := factoryBase.(ExporterFactory); ok {
		return factory.CreateHWExporter(), nil
	}
	return nil, fmt.Errorf("%v is not a valid hello world exporter factory", factoryBase)
}

// CreateProcessor ...
func (f *Factory) CreateProcessor(
	ctx context.Context,
	factoryBase component.ProcessorFactoryBase,
	logger *zap.Logger,
	nextConsumer component.Consumer,
	config configmodels.Processor,
) (component.Processor, error) {
	if factory, ok := factoryBase.(ProcessorFactory); ok {
		if nextConsumer, ok := nextConsumer.(Consumer); ok {
			return factory.CreateHWProcessor(nextConsumer), nil
		}
	}
	return nil, fmt.Errorf("%v is not a valid hello world processor factory", factoryBase)
}

// CreateFanOutConnector ...
func (f *Factory) CreateFanOutConnector(consumers []component.Consumer) component.Consumer {
	hwConsumers := make([]Consumer, len(consumers))
	for i, c := range consumers {
		hwConsumers[i] = c.(Consumer)
	}
	return newFanOutConnector(hwConsumers)
}

func newFanOutConnector(consumers []Consumer) Consumer {
	return fanOutConnector(consumers)
}

// CreateCloningFanOutConnector ...
func (f *Factory) CreateCloningFanOutConnector(consumers []component.Consumer) component.Consumer {
	hwConsumers := make([]Consumer, len(consumers))
	for i, c := range consumers {
		hwConsumers[i] = c.(Consumer)
	}
	return newCloningFanOutConnector(hwConsumers)
}

func newCloningFanOutConnector(consumers []Consumer) Consumer {
	return cloningFanOutConnector(consumers)
}

type fanOutConnector []Consumer

var _ Consumer = (*fanOutConnector)(nil)

func (fc fanOutConnector) ConsumeHW(hw HelloWorld) error {
	var errs []error
	for _, c := range fc {
		if err := c.ConsumeHW(hw); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

type cloningFanOutConnector []Consumer

var _ Consumer = (*cloningFanOutConnector)(nil)

func (tfc cloningFanOutConnector) ConsumeHW(hw HelloWorld) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone := hw.clone()
		if err := tfc[i].ConsumeHW(clone); err != nil {
			errs = append(errs, err)
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeHW(hw); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}
