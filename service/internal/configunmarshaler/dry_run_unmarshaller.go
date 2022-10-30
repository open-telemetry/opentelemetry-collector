package configunmarshaler

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
)

type ExtensionValidateError struct {
	Component string
	Id        config.ComponentID
	Err       string
}

type ExporterValidateError struct {
	Component string
	Id        config.ComponentID
	Err       string
}

type ReceiverValidateError struct {
	Component string
	Id        config.ComponentID
	Err       string
}

type ProcessorValidateError struct {
	Component string
	Id        config.ComponentID
	Err       string
}

var ExtensionValidateErrors []ExtensionValidateError
var ExportersValidateErrors []ExporterValidateError
var ReceiversValidateErrors []ReceiverValidateError
var ProcessorsValidateErrors []ProcessorValidateError

func (ConfigUnmarshaler) DryRunUnmarshal(v *confmap.Conf, factories component.Factories) (*config.Config, error, bool) {
	errorcheck := false
	if err := v.Unmarshal(&rawCfg, confmap.WithErrorUnused()); err != nil {
		return nil, configError{
			error: fmt.Errorf("error reading top level configuration sections: %w", err),
			code:  errUnmarshalTopLevelStructure,
		}, errorcheck
	}

	var cfg config.Config
	cfg.Extensions = validateUnmarshalExtensions(rawCfg.Extensions, factories.Extensions, &errorcheck)

	cfg.Receivers = validateUnmarshalReceivers(rawCfg.Receivers, factories.Receivers, &errorcheck)

	cfg.Processors = validateUnmarshalProcessors(rawCfg.Processors, factories.Processors, &errorcheck)

	cfg.Exporters = validateUnmarshalExporters(rawCfg.Exporters, factories.Exporters, &errorcheck)

	cfg.Service = rawCfg.Service

	return &cfg, nil, errorcheck
}

func validateUnmarshalExtensions(exts map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ExtensionFactory, errorcheck *bool) map[config.ComponentID]config.Extension {
	extensions := make(map[config.ComponentID]config.Extension)
	for id, value := range exts {
		factory, ok := factories[id.Type()]
		if !ok {
			*errorcheck = true
			err := ExtensionValidateError{
				Component: extensionsKeyName,
				Id:        id,
				Err:       fmt.Sprintf("unknown %s ptype %q for %q (valid values: %v)", extensionsKeyName, id.Type(), id, reflect.ValueOf(factories).MapKeys()),
			}
			ExtensionValidateErrors = append(ExtensionValidateErrors, err)
			continue
		}
		extensionCfg := factory.CreateDefaultConfig()
		extensionCfg.SetIDName(id.Name())
		if err := config.UnmarshalExtension(confmap.NewFromStringMap(value), extensionCfg); err != nil {
			*errorcheck = true
			err := ExtensionValidateError{
				Component: extensionsKeyName,
				Id:        id,
				Err:       fmt.Sprintf("error reading %s configuration for %q: %v", extensionsKeyName, id, err),
			}
			ExtensionValidateErrors = append(ExtensionValidateErrors, err)
			continue
		}
		extensions[id] = extensionCfg

	}
	return extensions
}

// LoadReceiver loads a receiver config from componentConfig using the provided factories.
func validateLoadReceiver(componentConfig *confmap.Conf, id config.ComponentID, factory component.ReceiverFactory, errorcheck *bool) (config.Receiver, error) {
	// Create the default config for this receiver.
	receiverCfg := factory.CreateDefaultConfig()
	receiverCfg.SetIDName(id.Name())

	// Now that the default config struct is created we can Unmarshal into it,
	// and it will apply user-defined config on top of the default.
	if err := config.UnmarshalReceiver(componentConfig, receiverCfg); err != nil {
		*errorcheck = true
		err := ReceiverValidateError{
			Component: receiversKeyName,
			Id:        id,
			Err:       fmt.Sprintf("error reading %s configuration for %q: %v", receiversKeyName, id, err),
		}
		ReceiversValidateErrors = append(ReceiversValidateErrors, err)
	}

	return receiverCfg, nil
}

func validateUnmarshalReceivers(recvs map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ReceiverFactory, errorcheck *bool) map[config.ComponentID]config.Receiver {
	// Prepare resulting map.
	receivers := make(map[config.ComponentID]config.Receiver)

	// Iterate over input map and create a config for each.
	for id, value := range recvs {
		// Find receiver factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			*errorcheck = true
			err := ReceiverValidateError{
				Component: receiversKeyName,
				Id:        id,
				Err:       fmt.Sprintf("unknown %s type %q for %q (valid values: %v)", receiversKeyName, id.Type(), id, reflect.ValueOf(factories).MapKeys()),
			}
			ReceiversValidateErrors = append(ReceiversValidateErrors, err)
			continue
		}

		receiverCfg, err := validateLoadReceiver(confmap.NewFromStringMap(value), id, factory, errorcheck)
		if err != nil {
			continue
		}

		receivers[id] = receiverCfg
	}

	return receivers
}

func validateUnmarshalExporters(exps map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ExporterFactory, errorcheck *bool) map[config.ComponentID]config.Exporter {
	// Prepare resulting map.
	exporters := make(map[config.ComponentID]config.Exporter)

	// Iterate over Exporters and create a config for each.
	for id, value := range exps {
		// Find exporter factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			*errorcheck = true
			err := ExporterValidateError{
				Component: exportersKeyName,
				Id:        id,
				Err:       fmt.Sprintf("unknown %s type %q for %q (valid values: %v)", exportersKeyName, id.Type(), id, reflect.ValueOf(factories).MapKeys()),
			}
			ExportersValidateErrors = append(ExportersValidateErrors, err)
			continue
		}

		// Create the default config for this exporter.
		exporterCfg := factory.CreateDefaultConfig()
		exporterCfg.SetIDName(id.Name())

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := config.UnmarshalExporter(confmap.NewFromStringMap(value), exporterCfg); err != nil {
			*errorcheck = true
			err := ExporterValidateError{
				Component: exportersKeyName,
				Id:        id,
				Err:       fmt.Sprintf("error reading %s configuration for %q: %v", exportersKeyName, id, err),
			}
			ExportersValidateErrors = append(ExportersValidateErrors, err)
			continue
		}

		exporters[id] = exporterCfg
	}

	return exporters
}

func validateUnmarshalProcessors(procs map[config.ComponentID]map[string]interface{}, factories map[config.Type]component.ProcessorFactory, errorcheck *bool) map[config.ComponentID]config.Processor {
	// Prepare resulting map.
	processors := make(map[config.ComponentID]config.Processor)

	// Iterate over processors and create a config for each.
	for id, value := range procs {
		// Find processor factory based on "type" that we read from config source.
		factory := factories[id.Type()]
		if factory == nil {
			*errorcheck = true
			err := ProcessorValidateError{
				Component: processorsKeyName,
				Id:        id,
				Err:       fmt.Sprintf("unknown %s type %q for %q (valid values: %v)", processorsKeyName, id.Type(), id, reflect.ValueOf(factories).MapKeys()),
			}
			ProcessorsValidateErrors = append(ProcessorsValidateErrors, err)
			continue
		}

		// Create the default config for this processor.
		processorCfg := factory.CreateDefaultConfig()
		processorCfg.SetIDName(id.Name())

		// Now that the default config struct is created we can Unmarshal into it,
		// and it will apply user-defined config on top of the default.
		if err := config.UnmarshalProcessor(confmap.NewFromStringMap(value), processorCfg); err != nil {
			*errorcheck = true
			err := ProcessorValidateError{
				Component: processorsKeyName,
				Id:        id,
				Err:       fmt.Sprintf("error reading %s configuration for %q: %v", processorsKeyName, id, err),
			}
			ProcessorsValidateErrors = append(ProcessorsValidateErrors, err)
			continue
		}

		processors[id] = processorCfg
	}

	return processors
}
func ReturnValidateErrors() ([]ExtensionValidateError, []ExporterValidateError, []ReceiverValidateError, []ProcessorValidateError) {
	return ExtensionValidateErrors, ExportersValidateErrors, ReceiversValidateErrors, ProcessorsValidateErrors

}
