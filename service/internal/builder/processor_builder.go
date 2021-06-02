package builder

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

type builtProcessor struct {
	tc consumer.Traces
	mc consumer.Metrics
	lc consumer.Logs

	nextTc *builtProcessor
	nextMc *builtProcessor
	nextLc *builtProcessor

	logger *zap.Logger

	mutatesData bool
	processor   component.Processor
}

type builtProcessors map[config.ComponentID]*builtProcessor

func (btProc *builtProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	return btProc.tc.ConsumeTraces(ctx, td)
}

func (btProc *builtProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	return btProc.mc.ConsumeMetrics(ctx, md)
}

func (btProc *builtProcessor) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	return btProc.lc.ConsumeLogs(ctx, ld)
}

func (btProc *builtProcessor) Start(ctx context.Context, host component.Host) error {
	return btProc.processor.Start(ctx, host)
}

func (btProc *builtProcessor) Shutdown(ctx context.Context) error {
	return btProc.processor.Shutdown(ctx)
}

func (btProc *builtProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{btProc.mutatesData}
}

type processorsBuilder struct {
	logger    *zap.Logger
	buildInfo component.BuildInfo
	config    *config.Config
	factories map[config.Type]component.ProcessorFactory
}

func (procBuilder *processorsBuilder) buildProcessors(ctx context.Context, dataType config.DataType, processors []config.ComponentID) (builtProcessors, error) {
	var err error
	btProcs := make(builtProcessors)
	for _, procId := range processors {
		procCfg := procBuilder.config.Processors[procId]
		factory := procBuilder.factories[procCfg.ID().Type()]

		componentLogger := procBuilder.logger.With(zap.String(zapKindKey, zapKindProcessor), zap.Stringer(zapNameKey, procCfg.ID()))
		creationParams := component.ProcessorCreateParams{
			Logger:    componentLogger,
			BuildInfo: procBuilder.buildInfo,
		}

		nextProc := &builtProcessor{}
		mutatesConsumedData := false
		switch dataType {
		case config.TracesDataType:
			var proc component.TracesProcessor
			proc, err = factory.CreateTracesProcessor(ctx, creationParams, procCfg, nextProc)
			if proc != nil {
				mutatesConsumedData = proc.Capabilities().MutatesData
			}
			btProc := &builtProcessor{proc, nil, nil, nextProc, nil, nil, creationParams.Logger, mutatesConsumedData, proc}
			btProcs[procId] = btProc
		case config.MetricsDataType:
			var proc component.MetricsProcessor
			proc, err = factory.CreateMetricsProcessor(ctx, creationParams, procCfg, nextProc)
			if proc != nil {
				mutatesConsumedData = proc.Capabilities().MutatesData
			}
			btProc := &builtProcessor{nil, proc, nil, nil, nextProc, nil, creationParams.Logger, mutatesConsumedData, proc}
			btProcs[procId] = btProc
		case config.LogsDataType:
			var proc component.LogsProcessor
			proc, err = factory.CreateLogsProcessor(ctx, creationParams, procCfg, nextProc)
			if proc != nil {
				mutatesConsumedData = proc.Capabilities().MutatesData
			}
			btProc := &builtProcessor{nil, nil, proc, nil, nil, nextProc, creationParams.Logger, mutatesConsumedData, proc}
			btProcs[procId] = btProc
		default:
			return nil, fmt.Errorf("error creating processor %q , data type %s is not supported",
				procId, dataType)
		}

		if err != nil {
			return nil, fmt.Errorf("error creating processor %q: %v", procId, err)
		}
	}

	return btProcs, nil
}

func BuildProcessors(
	logger *zap.Logger,
	buildInfo component.BuildInfo,
	config *config.Config,
	factories map[config.Type]component.ProcessorFactory,
	dataType config.DataType,
	processors []config.ComponentID,
) (builtProcessors, error) {
	procBuilder := &processorsBuilder{logger, buildInfo, config, factories}
	return procBuilder.buildProcessors(context.Background(), dataType, processors)
}
