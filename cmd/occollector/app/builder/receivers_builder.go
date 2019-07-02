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

package builder

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/internal"
	"github.com/open-telemetry/opentelemetry-service/models"
	"github.com/open-telemetry/opentelemetry-service/processor/multiconsumer"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

// builtReceiver is a receiver that is built based on a config. It can have
// a trace and/or a metrics component.
type builtReceiver struct {
	trace   receiver.TraceReceiver
	metrics receiver.MetricsReceiver
}

// Stop the receiver.
func (rcv *builtReceiver) Stop() error {
	var errors []error
	if rcv.trace != nil {
		err := rcv.trace.StopTraceReception()
		if err != nil {
			errors = append(errors, err)
		}
	}

	if rcv.metrics != nil {
		err := rcv.metrics.StopMetricsReception()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return internal.CombineErrors(errors)
}

// Start the receiver.
func (rcv *builtReceiver) Start(host receiver.Host) error {
	var errors []error
	if rcv.trace != nil {
		err := rcv.trace.StartTraceReception(host)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if rcv.metrics != nil {
		err := rcv.metrics.StartMetricsReception(host)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return internal.CombineErrors(errors)
}

// Receivers is a map of receivers created from receiver configs.
type Receivers map[models.Receiver]*builtReceiver

// StopAll stops all receivers.
func (rcvs Receivers) StopAll() {
	for _, rcv := range rcvs {
		rcv.Stop()
	}
}

// StartAll starts all receivers.
func (rcvs Receivers) StartAll(logger *zap.Logger, host receiver.Host) error {
	for cfg, rcv := range rcvs {
		logger.Info("Receiver is starting...", zap.String("receiver", cfg.Name()))

		if err := rcv.Start(host); err != nil {
			return err
		}
		logger.Info("Receiver is started.", zap.String("receiver", cfg.Name()))
	}
	return nil
}

// ReceiversBuilder builds receivers from config.
type ReceiversBuilder struct {
	logger             *zap.Logger
	config             *models.ConfigV2
	pipelineProcessors PipelineProcessors
}

// NewReceiversBuilder creates a new ReceiversBuilder. Call Build() on the returned value.
func NewReceiversBuilder(
	logger *zap.Logger,
	config *models.ConfigV2,
	pipelineProcessors PipelineProcessors,
) *ReceiversBuilder {
	return &ReceiversBuilder{logger, config, pipelineProcessors}
}

// Build receivers from config.
func (rb *ReceiversBuilder) Build() (Receivers, error) {
	receivers := make(Receivers)

	// Build receivers based on configuration.
	for _, cfg := range rb.config.Receivers {
		rcv, err := rb.buildReceiver(cfg)
		if err != nil {
			return nil, err
		}
		receivers[cfg] = rcv
	}

	return receivers, nil
}

// hasReceiver returns true if the pipeline is attached to specified receiver.
func hasReceiver(pipeline *models.Pipeline, receiverName string) bool {
	for _, name := range pipeline.Receivers {
		if name == receiverName {
			return true
		}
	}
	return false
}

type attachedPipelines map[models.DataType][]*builtProcessor

func (rb *ReceiversBuilder) findPipelinesToAttach(config models.Receiver) (attachedPipelines, error) {
	// A receiver may be attached to multiple pipelines. Pipelines may consume different
	// data types. We need to compile the list of pipelines of each type that must be
	// attached to this receiver according to configuration.

	pipelinesToAttach := make(attachedPipelines)
	pipelinesToAttach[models.TracesDataType] = make([]*builtProcessor, 0)
	pipelinesToAttach[models.MetricsDataType] = make([]*builtProcessor, 0)

	// Iterate over all pipelines.
	for _, pipelineCfg := range rb.config.Pipelines {
		// Get the first processor of the pipeline.
		pipelineProcessor := rb.pipelineProcessors[pipelineCfg]
		if pipelineProcessor == nil {
			return nil, fmt.Errorf("cannot find pipeline processor for pipeline %s",
				pipelineCfg.Name)
		}

		// Is this receiver attached to the pipeline?
		if hasReceiver(pipelineCfg, config.Name()) {
			// Yes, add it to the list of pipelines of corresponding data type.
			pipelinesToAttach[pipelineCfg.InputType] =
				append(pipelinesToAttach[pipelineCfg.InputType], pipelineProcessor)
		}
	}

	return pipelinesToAttach, nil
}

func (rb *ReceiversBuilder) attachReceiverToPipelines(
	factory receiver.Factory,
	dataType models.DataType,
	config models.Receiver,
	rcv *builtReceiver,
	pipelineProcessors []*builtProcessor,
) error {
	// There are pipelines of the specified data type that must be attached to
	// the receiver. Create the receiver of corresponding data type and make
	// sure its output is fanned out to all attached pipelines.
	var err error
	switch dataType {
	case models.TracesDataType:
		// First, create the fan out junction point.
		junction := buildFanoutTraceConsumer(pipelineProcessors)

		// Now create the receiver and tell it to send to the junction point.
		rcv.trace, err = factory.CreateTraceReceiver(context.Background(), rb.logger, config, junction)

	case models.MetricsDataType:
		junction := buildFanoutMetricConsumer(pipelineProcessors)
		rcv.metrics, err = factory.CreateMetricsReceiver(rb.logger, config, junction)
	}

	if err != nil {
		if err == models.ErrDataTypeIsNotSupported {
			return fmt.Errorf(
				"receiver %s does not support %s but it was used in a "+
					"%s pipeline",
				config.Name(),
				dataType.GetString(),
				dataType.GetString())
		}
		return fmt.Errorf("cannot create receiver %s", config.Name())
	}

	rb.logger.Info("Receiver is enabled.",
		zap.String("receiver", config.Name()), zap.String("datatype", dataType.GetString()))

	return nil
}

func (rb *ReceiversBuilder) buildReceiver(config models.Receiver) (*builtReceiver, error) {

	// First find pipelines that must be attached to this receiver.
	pipelinesToAttach, err := rb.findPipelinesToAttach(config)
	if err != nil {
		return nil, err
	}

	// Prepare to build the receiver.
	factory := receiver.GetReceiverFactory(config.Type())
	rcv := &builtReceiver{}

	// Now we have list of pipelines broken down by data type. Iterate for each data type.
	for dataType, pipelines := range pipelinesToAttach {
		if len(pipelines) == 0 {
			// No pipelines of this data type are attached to this receiver.
			continue
		}

		// Attach the corresponding part of the receiver to all pipelines that require
		// this data type.
		err := rb.attachReceiverToPipelines(factory, dataType, config, rcv, pipelines)
		if err != nil {
			return nil, err
		}
	}

	return rcv, nil
}

func buildFanoutTraceConsumer(pipelineFrontProcessors []*builtProcessor) consumer.TraceConsumer {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelineFrontProcessors) == 1 {
		return pipelineFrontProcessors[0].tc
	}

	var pipelineConsumers []consumer.TraceConsumer
	for _, builtProc := range pipelineFrontProcessors {
		pipelineConsumers = append(pipelineConsumers, builtProc.tc)
	}

	// Create a junction point that fans out to all pipelines.
	return multiconsumer.NewTraceProcessor(pipelineConsumers)
}

func buildFanoutMetricConsumer(pipelineFrontProcessors []*builtProcessor) consumer.MetricsConsumer {
	// Optimize for the case when there is only one processor, no need to create junction point.
	if len(pipelineFrontProcessors) == 1 {
		return pipelineFrontProcessors[0].mc
	}

	var pipelineConsumers []consumer.MetricsConsumer
	for _, builtProc := range pipelineFrontProcessors {
		pipelineConsumers = append(pipelineConsumers, builtProc.mc)
	}

	// Create a junction point that fans out to all pipelines.
	return multiconsumer.NewMetricsProcessor(pipelineConsumers)
}
