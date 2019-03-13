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

package collector

import (
	"fmt"
	"os"
	"time"

	tchReporter "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/tchannel"
	"github.com/spf13/viper"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/builder"
	"github.com/census-instrumentation/opencensus-service/cmd/occollector/app/sender"
	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/exporter/loggingexporter"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor/nodebatcher"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor/queued"
	"github.com/census-instrumentation/opencensus-service/internal/collector/processor/tailsampling"
	"github.com/census-instrumentation/opencensus-service/internal/collector/sampling"
	"github.com/census-instrumentation/opencensus-service/internal/config"
	"github.com/census-instrumentation/opencensus-service/processor/multiconsumer"
)

func createExporters(v *viper.Viper, logger *zap.Logger) ([]func(), []consumer.TraceConsumer, []consumer.MetricsConsumer) {
	// TODO: (@pjanotti) this is slightly modified from agent but in the end duplication, need to consolidate style and visibility.
	traceExporters, metricsExporters, doneFns, err := config.ExportersFromViperConfig(logger, v)
	if err != nil {
		logger.Fatal("Failed to create config for exporters", zap.Error(err))
	}

	wrappedDoneFns := make([]func(), 0, len(doneFns))
	for _, doneFn := range doneFns {
		wrapperFn := func() {
			if err := doneFn(); err != nil {
				logger.Warn("Error when closing exporters", zap.Error(err))
			}
		}

		wrappedDoneFns = append(wrappedDoneFns, wrapperFn)
	}

	return wrappedDoneFns, traceExporters, metricsExporters
}

func buildQueuedSpanProcessor(
	logger *zap.Logger, opts *builder.QueuedSpanProcessorCfg,
) (closeFns []func(), queuedSpanProcessor consumer.TraceConsumer, err error) {
	logger.Info("Constructing queue processor with name", zap.String("name", opts.Name))

	// build span batch sender from configured options
	var spanSender consumer.TraceConsumer
	switch opts.SenderType {
	case builder.ThriftTChannelSenderType:
		logger.Info("Initializing thrift-tChannel sender")
		thriftTChannelSenderOpts := opts.SenderConfig.(*builder.JaegerThriftTChannelSenderCfg)
		tchrepbuilder := &tchReporter.Builder{
			CollectorHostPorts: thriftTChannelSenderOpts.CollectorHostPorts,
			DiscoveryMinPeers:  thriftTChannelSenderOpts.DiscoveryMinPeers,
			ConnCheckTimeout:   thriftTChannelSenderOpts.DiscoveryConnCheckTimeout,
		}
		tchreporter, err := tchrepbuilder.CreateReporter(metrics.NullFactory, logger)
		if err != nil {
			logger.Fatal("Cannot create tchannel reporter.", zap.Error(err))
			return nil, nil, err
		}
		spanSender = sender.NewJaegerThriftTChannelSender(tchreporter, logger)
	case builder.ThriftHTTPSenderType:
		thriftHTTPSenderOpts := opts.SenderConfig.(*builder.JaegerThriftHTTPSenderCfg)
		logger.Info("Initializing thrift-HTTP sender",
			zap.String("url", thriftHTTPSenderOpts.CollectorEndpoint))
		spanSender = sender.NewJaegerThriftHTTPSender(
			thriftHTTPSenderOpts.CollectorEndpoint,
			thriftHTTPSenderOpts.Headers,
			logger,
			sender.HTTPTimeout(thriftHTTPSenderOpts.Timeout),
		)
	}
	doneFns, traceExporters, _ := createExporters(opts.RawConfig, logger)

	if spanSender == nil && len(traceExporters) == 0 {
		if opts.SenderType != "" {
			logger.Fatal("Unrecognized sender type", zap.String("SenderType", string(opts.SenderType)))
		}
		logger.Fatal("No senders or exporters configured.")
	}

	allSendersAndExporters := make([]consumer.TraceConsumer, 0, 1+len(traceExporters))
	if spanSender != nil {
		allSendersAndExporters = append(allSendersAndExporters, spanSender)
	}
	for _, traceExporter := range traceExporters {
		allSendersAndExporters = append(allSendersAndExporters, traceExporter)
	}

	var batchingOptions []nodebatcher.Option
	if opts.BatchingConfig.Enable {
		cfg := opts.BatchingConfig
		if cfg.Timeout != nil {
			batchingOptions = append(batchingOptions, nodebatcher.WithTimeout(*cfg.Timeout))
		}
		if cfg.NumTickers > 0 {
			batchingOptions = append(
				batchingOptions, nodebatcher.WithNumTickers(cfg.NumTickers),
			)
		}
		if cfg.TickTime != nil {
			batchingOptions = append(
				batchingOptions, nodebatcher.WithTickTime(*cfg.TickTime),
			)
		}
		if cfg.SendBatchSize != nil {
			batchingOptions = append(
				batchingOptions, nodebatcher.WithSendBatchSize(*cfg.SendBatchSize),
			)
		}
		if cfg.RemoveAfterTicks != nil {
			batchingOptions = append(
				batchingOptions, nodebatcher.WithRemoveAfterTicks(*cfg.RemoveAfterTicks),
			)
		}
	}

	queuedConsumers := make([]consumer.TraceConsumer, 0, len(allSendersAndExporters))
	for _, senderOrExporter := range allSendersAndExporters {
		// build queued span processor with underlying sender
		queuedConsumers = append(
			queuedConsumers,
			queued.NewQueuedSpanProcessor(
				senderOrExporter,
				queued.Options.WithLogger(logger),
				queued.Options.WithName(opts.Name),
				queued.Options.WithNumWorkers(opts.NumWorkers),
				queued.Options.WithQueueSize(opts.QueueSize),
				queued.Options.WithRetryOnProcessingFailures(opts.RetryOnFailure),
				queued.Options.WithBackoffDelay(opts.BackoffDelay),
				queued.Options.WithBatching(opts.BatchingConfig.Enable),
				queued.Options.WithBatchingOptions(batchingOptions...),
			),
		)
	}
	return doneFns, processor.NewMultiSpanProcessor(queuedConsumers), nil
}

func buildSamplingProcessor(cfg *builder.SamplingCfg, nameToTraceConsumer map[string]consumer.TraceConsumer, v *viper.Viper, logger *zap.Logger) (consumer.TraceConsumer, error) {
	var policies []*tailsampling.Policy
	seenExporter := make(map[string]bool)
	for _, polCfg := range cfg.Policies {
		policy := &tailsampling.Policy{
			Name: string(polCfg.Name),
		}

		// As the number of sampling policies grow this should be changed to a map.
		switch polCfg.Type {
		case builder.AlwaysSample:
			policy.Evaluator = sampling.NewAlwaysSample()
		case builder.NumericTagFilter:
			numTagFilterCfg := polCfg.Configuration.(*builder.NumericTagFilterCfg)
			policy.Evaluator = sampling.NewNumericTagFilter(numTagFilterCfg.Tag, numTagFilterCfg.MinValue, numTagFilterCfg.MaxValue)
		case builder.StringTagFilter:
			strTagFilterCfg := polCfg.Configuration.(*builder.StringTagFilterCfg)
			policy.Evaluator = sampling.NewStringTagFilter(strTagFilterCfg.Tag, strTagFilterCfg.Values)
		case builder.RateLimiting:
			rateLimitingCfg := polCfg.Configuration.(*builder.RateLimitingCfg)
			policy.Evaluator = sampling.NewRateLimiting(rateLimitingCfg.SpansPerSecond)
		default:
			return nil, fmt.Errorf("unknown sampling policy %s", polCfg.Name)
		}

		var policyProcessors []consumer.TraceConsumer
		for _, exporter := range polCfg.Exporters {
			if _, ok := seenExporter[exporter]; ok {
				return nil, fmt.Errorf("multiple sampling polices pointing to exporter %q", exporter)
			}
			seenExporter[exporter] = true

			policyProcessor, ok := nameToTraceConsumer[exporter]
			if !ok {
				return nil, fmt.Errorf("invalid exporter %q for sampling policy %q", exporter, polCfg.Name)
			}

			policyProcessors = append(policyProcessors, policyProcessor)
		}

		numPolicyProcessors := len(policyProcessors)
		switch {
		case numPolicyProcessors == 1:
			policy.Destination = policyProcessors[0]
		case numPolicyProcessors > 1:
			policy.Destination = processor.NewMultiSpanProcessor(policyProcessors)
		default:
			return nil, fmt.Errorf("no exporters for sampling policy %q", polCfg.Name)
		}

		policies = append(policies, policy)
	}

	if len(policies) < 1 {
		return nil, fmt.Errorf("no sampling policies were configured")
	}

	tailCfg := builder.NewDefaultTailBasedCfg().InitFromViper(v)
	tailSamplingProcessor, err := tailsampling.NewTailSamplingSpanProcessor(
		policies,
		tailCfg.NumTraces,
		128,
		tailCfg.DecisionWait,
		logger)
	return tailSamplingProcessor, err
}

func startProcessor(v *viper.Viper, logger *zap.Logger) (consumer.TraceConsumer, []func()) {
	// Build pipeline from its end: 1st exporters, the OC-proto queue processor, and
	// finally the receivers.
	var closeFns []func()
	var traceConsumers []consumer.TraceConsumer
	nameToTraceConsumer := make(map[string]consumer.TraceConsumer)
	exportersCloseFns, traceExporters, metricsExporters := createExporters(v, logger)
	closeFns = append(closeFns, exportersCloseFns...)
	if len(traceExporters) > 0 {
		// Exporters need an extra hop from OC-proto to span data: to workaround that for now
		// we will use a special processor that transforms the data to a format that they can consume.
		// TODO: (@pjanotti) we should avoid this step in the long run, its an extra hop just to re-use
		// the exporters: this can lose node information and it is not ideal for performance and delegates
		// the retry/buffering to the exporters (that are designed to run within the tracing process).
		traceExpProc := multiconsumer.NewTraceProcessor(traceExporters)
		nameToTraceConsumer["exporters"] = traceExpProc
		traceConsumers = append(traceConsumers, traceExpProc)
	}

	// TODO: (@pjanotti) make use of metrics exporters
	_ = metricsExporters

	if builder.LoggingExporterEnabled(v) {
		dbgProc, _ := loggingexporter.NewTraceExporter(logger)
		// TODO: Add this to the exporters list and avoid treating it specially. Don't know all the implications.
		nameToTraceConsumer["debug"] = dbgProc
		traceConsumers = append(traceConsumers, dbgProc)
	}

	multiProcessorCfg := builder.NewDefaultMultiSpanProcessorCfg().InitFromViper(v)
	for _, queuedJaegerProcessorCfg := range multiProcessorCfg.Processors {
		logger.Info("Queued Jaeger Sender Enabled")
		doneFns, queuedJaegerProcessor, err := buildQueuedSpanProcessor(logger, queuedJaegerProcessorCfg)
		if err != nil {
			logger.Error("Failed to build the queued span processor", zap.Error(err))
			os.Exit(1)
		}
		nameToTraceConsumer[queuedJaegerProcessorCfg.Name] = queuedJaegerProcessor
		traceConsumers = append(traceConsumers, queuedJaegerProcessor)
		closeFns = append(closeFns, doneFns...)
	}

	if len(traceConsumers) == 0 {
		logger.Warn("Nothing to do: no processor was enabled. Shutting down.")
		os.Exit(1)
	}

	var tailSamplingProcessor consumer.TraceConsumer
	samplingProcessorCfg := builder.NewDefaultSamplingCfg().InitFromViper(v)
	if samplingProcessorCfg.Mode == builder.TailSampling {
		var err error
		tailSamplingProcessor, err = buildSamplingProcessor(samplingProcessorCfg, nameToTraceConsumer, v, logger)
		if err != nil {
			logger.Error("Falied to build the sampling processor", zap.Error(err))
			os.Exit(1)
		}
	} else if builder.DebugTailSamplingEnabled(v) {
		policy := []*tailsampling.Policy{
			{
				Name:        "tail-always-sampling",
				Evaluator:   sampling.NewAlwaysSample(),
				Destination: processor.NewMultiSpanProcessor(traceConsumers),
			},
		}
		var err error
		tailSamplingProcessor, err = tailsampling.NewTailSamplingSpanProcessor(policy, 50000, 128, 10*time.Second, logger)
		if err != nil {
			logger.Error("Falied to build the debug tail-sampling processor", zap.Error(err))
			os.Exit(1)
		}
		logger.Info("Debugging tail-sampling with always sample policy (num_traces: 50000; decision_wait: 10s)")
	}

	if tailSamplingProcessor != nil {
		// SpanProcessors are going to go all via the tail sampling processor.
		traceConsumers = []consumer.TraceConsumer{tailSamplingProcessor}
	}

	// Wraps processors in a single one to be connected to all enabled receivers.
	var processorOptions []processor.MultiProcessorOption
	if multiProcessorCfg.Global != nil && multiProcessorCfg.Global.Attributes != nil {
		logger.Info(
			"Found global attributes config",
			zap.Bool("overwrite", multiProcessorCfg.Global.Attributes.Overwrite),
			zap.Any("values", multiProcessorCfg.Global.Attributes.Values),
		)
		processorOptions = append(
			processorOptions,
			processor.WithAddAttributes(
				multiProcessorCfg.Global.Attributes.Values,
				multiProcessorCfg.Global.Attributes.Overwrite,
			),
		)
	}
	return processor.NewMultiSpanProcessor(traceConsumers, processorOptions...), closeFns
}
