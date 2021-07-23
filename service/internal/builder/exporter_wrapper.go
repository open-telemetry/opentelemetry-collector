package builder

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

type exporterWrapper struct {
	inputType config.DataType
	mc        consumer.Metrics
	tc        consumer.Traces
	lc        consumer.Logs
	exporter  component.Exporter
	id        config.ComponentID
	set       component.ExporterCreateSettings
	factory   component.ExporterFactory
}

var errDataTypeNotSupported error = errors.New("dataType not supported")

func (wrapper *exporterWrapper) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	if wrapper.inputType != config.LogsDataType {
		return errDataTypeNotSupported
	} else {
		return wrapper.lc.ConsumeLogs(ctx, ld)
	}
}

func (wrapper *exporterWrapper) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	if wrapper.inputType != config.TracesDataType {
		return errDataTypeNotSupported
	} else {
		return wrapper.tc.ConsumeTraces(ctx, td)
	}
}

func (wrapper *exporterWrapper) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	if wrapper.inputType != config.MetricsDataType {
		return errDataTypeNotSupported
	} else {
		return wrapper.mc.ConsumeMetrics(ctx, md)
	}
}

func (wrapper *exporterWrapper) Capabilities() consumer.Capabilities {
	switch wrapper.inputType {
	case config.LogsDataType:
		return wrapper.lc.Capabilities()
	case config.MetricsDataType:
		return wrapper.mc.Capabilities()
	case config.TracesDataType:
		return wrapper.Capabilities()
	}
	return consumer.Capabilities{}
}

func (wrapper *exporterWrapper) Start(ctx context.Context, host component.Host) error {
	return wrapper.exporter.Start(ctx, host)
}

func (wrapper *exporterWrapper) Shutdown(ctx context.Context) error {
	return wrapper.exporter.Shutdown(ctx)
}

func (wrapper *exporterWrapper) Reload(host component.Host, ctx context.Context, cfg interface{}) error {
	expCfg, ok := cfg.(config.Exporter)

	if !ok {
		return fmt.Errorf("error when reload exporter:%q for invalid cfg:%v", wrapper.id, cfg)
	}

	if expCfg.ID() != wrapper.id {
		return fmt.Errorf("error when reload exporter:%q for invalid conf id:%v", wrapper.id, expCfg.ID())
	}

	if reloadableExp, ok := wrapper.exporter.(component.Reloadable); ok {
		return reloadableExp.Reload(host, ctx, cfg)
	}

	oldExp := wrapper.exporter

	var err error
	switch wrapper.inputType {
	case config.MetricsDataType:
		var exp component.MetricsExporter
		exp, err = wrapper.factory.CreateMetricsExporter(ctx, wrapper.set, expCfg)
		if exp != nil && err == nil {
			err = exp.Start(ctx, host)
		}

		if exp == nil {
			return err
		}

		if exp != nil && err == nil {
			wrapper.exporter = exp
			wrapper.mc = exp
		}
	case config.LogsDataType:
		var exp component.LogsExporter
		exp, err = wrapper.factory.CreateLogsExporter(ctx, wrapper.set, expCfg)
		if exp != nil && err == nil {
			err = exp.Start(ctx, host)
		}

		if exp == nil {
			return err
		}

		if exp != nil && err == nil {
			wrapper.exporter = exp
			wrapper.lc = exp
		}
	case config.TracesDataType:
		var exp component.TracesExporter
		exp, err = wrapper.factory.CreateTracesExporter(ctx, wrapper.set, expCfg)
		if exp != nil && err == nil {
			err = exp.Start(ctx, host)
		}

		if exp == nil {
			return err
		}

		if exp != nil && err == nil {
			wrapper.exporter = exp
			wrapper.tc = exp
		}
	}

	if err != nil {
		return err
	}

	if err := oldExp.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
