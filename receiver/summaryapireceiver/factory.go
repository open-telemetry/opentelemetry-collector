package summaryapireceiver

import (
	"context"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/internal/configmodels"
	"github.com/open-telemetry/opentelemetry-service/internal/factories"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// This file implements config V2 for SummaryApi receiver.

var _ = factories.RegisterReceiverFactory(&ReceiverFactory{})

const (
	// The value of "type" key in configuration.
	typeStr = "summaryapi"

	defaultBindEndpoint = "127.0.0.1:9090"
)

// ReceiverFactory is the factory for receiver.
type ReceiverFactory struct {
}

// Type gets the type of the Receiver config created by this factory.
func (f *ReceiverFactory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this config.
func (f *ReceiverFactory) CustomUnmarshaler() factories.CustomUnmarshaler {
	return nil
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *ReceiverFactory) CreateDefaultConfig() configmodels.Receiver {
	return &ConfigV2{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  typeStr,
			Endpoint: defaultBindEndpoint,
		},
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *ReceiverFactory) CreateTraceReceiver(
	ctx context.Context,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (receiver.TraceReceiver, error) {
	// SummaryApi does not support traces
	return nil, factories.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *ReceiverFactory) CreateMetricsReceiver(
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (receiver.MetricsReceiver, error) {

	rCfg := cfg.(*ConfigV2)
	sac, err := NewSummaryApiCollector(rCfg.scrapeInterval, rCfg.kubeletEndpoint, rCfg.metricPrefix, consumer)
	if err != nil {
		return nil, err
	}
	sar := &Receiver{
		sac: sac,
	}
	return sar, nil
}
