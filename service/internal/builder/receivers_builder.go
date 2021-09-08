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

package builder

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

var errUnusedReceiver = errors.New("receiver defined but not used by any pipeline")

// builtReceiver is a receiver that is built based on a config. It can have
// a trace and/or a metrics component.
type builtReceiver struct {
	logger   *zap.Logger
	receiver component.Receiver
}

// Start starts the receiver.
func (rcv *builtReceiver) Start(ctx context.Context, host component.Host) error {
	return rcv.receiver.Start(ctx, host)
}

// Shutdown stops the receiver.
func (rcv *builtReceiver) Shutdown(ctx context.Context) error {
	return rcv.receiver.Shutdown(ctx)
}

// Receivers is a map of receivers created from receiver configs.
type Receivers map[config.ComponentID]*builtReceiver

// ShutdownAll stops all receivers.
func (rcvs Receivers) ShutdownAll(ctx context.Context) error {
	var errs []error
	for _, rcv := range rcvs {
		err := rcv.Shutdown(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return consumererror.Combine(errs)
}

// StartAll starts all receivers.
func (rcvs Receivers) StartAll(ctx context.Context, host component.Host) error {
	for _, rcv := range rcvs {
		rcv.logger.Info("Receiver is starting...")

		if err := rcv.Start(ctx, newHostWrapper(host, rcv.logger)); err != nil {
			return err
		}
		rcv.logger.Info("Receiver started.")
	}
	return nil
}

// receiversBuilder builds receivers from config.
type receiversBuilder struct {
	config         *config.Config
	builtPipelines BuiltPipelines
	factories      map[config.Type]component.ReceiverFactory
}

// BuildReceivers builds Receivers from config.
func BuildReceivers(
	logger *zap.Logger,
	tracerProvider trace.TracerProvider,
	buildInfo component.BuildInfo,
	cfg *config.Config,
	builtPipelines BuiltPipelines,
	factories map[config.Type]component.ReceiverFactory,
) (Receivers, error) {
	rb := &receiversBuilder{cfg, builtPipelines, factories}

	receivers := make(Receivers)
	for recvID, recvCfg := range cfg.Receivers {
		set := component.ReceiverCreateSettings{
			TelemetryCreateSettings: component.TelemetryCreateSettings{
				Logger:         logger.With(zap.String(zapKindKey, zapKindReceiver), zap.String(zapNameKey, recvID.String())),
				TracerProvider: tracerProvider,
			},
			BuildInfo: buildInfo,
		}

		rcv, err := rb.buildReceiver(context.Background(), set, recvCfg)
		if err != nil {
			if err == errUnusedReceiver {
				set.Logger.Info("Ignoring receiver as it is not used by any pipeline")
				continue
			}
			return nil, err
		}
		receivers[recvID] = rcv
	}

	return receivers, nil
}

// hasReceiver returns true if the pipeline is attached to specified receiver.
func hasReceiver(pipeline *config.Pipeline, receiverID config.ComponentID) bool {
	for _, id := range pipeline.Receivers {
		if id == receiverID {
			return true
		}
	}
	return false
}

type attachedPipelines map[config.DataType][]*builtPipeline

func (rb *receiversBuilder) findPipelinesToAttach(receiverID config.ComponentID) (attachedPipelines, error) {
	// A receiver may be attached to multiple pipelines. Pipelines may consume different
	// data types. We need to compile the list of pipelines of each type that must be
	// attached to this receiver according to configuration.

	pipelinesToAttach := make(attachedPipelines)
	pipelinesToAttach[config.TracesDataType] = make([]*builtPipeline, 0)
	pipelinesToAttach[config.MetricsDataType] = make([]*builtPipeline, 0)

	// Iterate over all pipelines.
	for _, pipelineCfg := range rb.config.Service.Pipelines {
		// Get the first processor of the pipeline.
		pipelineProcessor := rb.builtPipelines[pipelineCfg]
		if pipelineProcessor == nil {
			return nil, fmt.Errorf("cannot find pipeline processor for pipeline %s",
				pipelineCfg.Name)
		}

		// Is this receiver attached to the pipeline?
		if hasReceiver(pipelineCfg, receiverID) {
			if _, exists := pipelinesToAttach[pipelineCfg.InputType]; !exists {
				pipelinesToAttach[pipelineCfg.InputType] = make([]*builtPipeline, 0)
			}

			// Yes, add it to the list of pipelines of corresponding data type.
			pipelinesToAttach[pipelineCfg.InputType] =
				append(pipelinesToAttach[pipelineCfg.InputType], pipelineProcessor)
		}
	}

	return pipelinesToAttach, nil
}

func attachReceiverToPipelines(
	ctx context.Context,
	set component.ReceiverCreateSettings,
	factory component.ReceiverFactory,
	dataType config.DataType,
	cfg config.Receiver,
	rcv *builtReceiver,
	builtPipelines []*builtPipeline,
) error {
	// There are pipelines of the specified data type that must be attached to
	// the receiver. Create the receiver of corresponding data type and make
	// sure its output is fanned out to all attached pipelines.
	var err error
	var createdReceiver component.Receiver

	switch dataType {
	case config.TracesDataType:
		junction := buildFanoutTraceConsumer(builtPipelines)
		createdReceiver, err = factory.CreateTracesReceiver(ctx, set, cfg, junction)

	case config.MetricsDataType:
		junction := buildFanoutMetricConsumer(builtPipelines)
		createdReceiver, err = factory.CreateMetricsReceiver(ctx, set, cfg, junction)

	case config.LogsDataType:
		junction := buildFanoutLogConsumer(builtPipelines)
		createdReceiver, err = factory.CreateLogsReceiver(ctx, set, cfg, junction)

	default:
		err = componenterror.ErrDataTypeIsNotSupported
	}

	if err != nil {
		if err == componenterror.ErrDataTypeIsNotSupported {
			return fmt.Errorf(
				"receiver %v does not support %s but it was used in a %s pipeline",
				cfg.ID(), dataType, dataType)
		}
		return fmt.Errorf("cannot create receiver %v: %w", cfg.ID(), err)
	}

	// Check if the factory really created the receiver.
	if createdReceiver == nil {
		return fmt.Errorf("factory for %v produced a nil receiver", cfg.ID())
	}

	if rcv.receiver != nil {
		// The receiver was previously created for this config. This can happen if the
		// same receiver type supports more than one data type. In that case we expect
		// that CreateTracesReceiver and CreateMetricsReceiver return the same value.
		if rcv.receiver != createdReceiver {
			return fmt.Errorf(
				"factory for %v is implemented incorrectly: "+
					"CreateTracesReceiver and CreateMetricsReceiver must return the same "+
					"receiver pointer when creating receivers of different data types",
				cfg.ID(),
			)
		}
	}
	rcv.receiver = createdReceiver

	set.Logger.Info("Receiver was built.", zap.String("datatype", string(dataType)))

	return nil
}

func (rb *receiversBuilder) buildReceiver(ctx context.Context, set component.ReceiverCreateSettings, cfg config.Receiver) (*builtReceiver, error) {

	// First find pipelines that must be attached to this receiver.
	pipelinesToAttach, err := rb.findPipelinesToAttach(cfg.ID())
	if err != nil {
		return nil, err
	}

	// Prepare to build the receiver.
	factory := rb.factories[cfg.ID().Type()]
	if factory == nil {
		return nil, fmt.Errorf("receiver factory not found for: %v", cfg.ID())
	}
	rcv := &builtReceiver{
		logger: set.Logger,
	}

	// Now we have list of pipelines broken down by data type. Iterate for each data type.
	for dataType, pipelines := range pipelinesToAttach {
		if len(pipelines) == 0 {
			// No pipelines of this data type are attached to this receiver.
			continue
		}

		// Attach the corresponding part of the receiver to all pipelines that require
		// this data type.
		err := attachReceiverToPipelines(ctx, set, factory, dataType, cfg, rcv, pipelines)
		if err != nil {
			return nil, err
		}
	}

	if rcv.receiver == nil {
		return nil, errUnusedReceiver
	}

	return rcv, nil
}

func buildFanoutTraceConsumer(pipelines []*builtPipeline) consumer.Traces {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelines) == 1 {
		return pipelines[0].firstTC
	}

	var pipelineConsumers []consumer.Traces
	anyPipelineMutatesData := false
	for _, pipeline := range pipelines {
		pipelineConsumers = append(pipelineConsumers, pipeline.firstTC)
		anyPipelineMutatesData = anyPipelineMutatesData || pipeline.MutatesData
	}

	// Create a junction point that fans out to all pipelines.
	if anyPipelineMutatesData {
		// If any pipeline mutates data use a cloning fan out connector
		// so that it is safe to modify fanned out data.
		// TODO: if there are more than 2 pipelines only clone data for pipelines that
		// declare the intent to mutate the data. Pipelines that do not mutate the data
		// can consume shared data.
		return fanoutconsumer.NewTracesCloning(pipelineConsumers)
	}
	return fanoutconsumer.NewTraces(pipelineConsumers)
}

func buildFanoutMetricConsumer(pipelines []*builtPipeline) consumer.Metrics {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelines) == 1 {
		return pipelines[0].firstMC
	}

	var pipelineConsumers []consumer.Metrics
	anyPipelineMutatesData := false
	for _, pipeline := range pipelines {
		pipelineConsumers = append(pipelineConsumers, pipeline.firstMC)
		anyPipelineMutatesData = anyPipelineMutatesData || pipeline.MutatesData
	}

	// Create a junction point that fans out to all pipelines.
	if anyPipelineMutatesData {
		// If any pipeline mutates data use a cloning fan out connector
		// so that it is safe to modify fanned out data.
		// TODO: if there are more than 2 pipelines only clone data for pipelines that
		// declare the intent to mutate the data. Pipelines that do not mutate the data
		// can consume shared data.
		return fanoutconsumer.NewMetricsCloning(pipelineConsumers)
	}
	return fanoutconsumer.NewMetrics(pipelineConsumers)
}

func buildFanoutLogConsumer(pipelines []*builtPipeline) consumer.Logs {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelines) == 1 {
		return pipelines[0].firstLC
	}

	var pipelineConsumers []consumer.Logs
	anyPipelineMutatesData := false
	for _, pipeline := range pipelines {
		pipelineConsumers = append(pipelineConsumers, pipeline.firstLC)
		anyPipelineMutatesData = anyPipelineMutatesData || pipeline.MutatesData
	}

	// Create a junction point that fans out to all pipelines.
	if anyPipelineMutatesData {
		// If any pipeline mutates data use a cloning fan out connector
		// so that it is safe to modify fanned out data.
		// TODO: if there are more than 2 pipelines only clone data for pipelines that
		// declare the intent to mutate the data. Pipelines that do not mutate the data
		// can consume shared data.
		return fanoutconsumer.NewLogsCloning(pipelineConsumers)
	}
	return fanoutconsumer.NewLogs(pipelineConsumers)
}
