package avitoquotaprocessor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/generated/rpc/clients/paas_observability"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/limiter"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/quota_processor"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/rules"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/syncer"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		CreateDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

func CreateDefaultConfig() component.Config {
	return &Config{
		RetryRulesOnStartCount: 20,
		StartWithoutRules:      true,
		RulesUpdateInterval:    time.Minute,
		RateLimiterDryRun:      true,
	}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	config, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Errorf("invalid config type, expected %T", Config{})
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	var level zapcore.Level
	err := level.Set(config.LogLevel)
	if err != nil {
		set.Logger.With(zap.Error(err), zap.String("level", config.LogLevel)).Error("specified invalid log level")
		return nil, errors.Wrap(err, "invalid log level")
	}
	logger := set.Logger.WithOptions(zap.IncreaseLevel(level))

	telemetry, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		set.Logger.With(zap.Error(err)).Error("failed to create telemetry")
		return nil, errors.Wrap(err, "failed to create telemetry")
	}
	paasObservabilityClient := paas_observability.New(rpcClient(paas_observability.Name, config.ClientTls))

	clientID := uuid.New().String()
	// pg
	rulesPgSyncer := syncer.NewPostgresSyncLimiter(logger, telemetry, paasObservabilityClient, clientID, config.RulesUpdateInterval)
	rulesPgSyncer.Start(ctx)

	initRules := rules.Rules{}
	var errRules error
	for i := 0; i < config.RetryRulesOnStartCount; i++ {
		initRules, errRules = rulesPgSyncer.GetRules(ctx)
		if errRules == nil {
			break
		}
	}
	if errRules != nil {
		if config.StartWithoutRules {
			logger.Warn("failed to get rules on start, starting without rules", zap.Error(errRules))
		} else {
			logger.Error("failed to get rules on start", zap.Error(errRules))
			return nil, err
		}
	}

	rulesLimiter := limiter.NewShardedManager(
		telemetry,
		logger,
		config.RateLimiterDryRun,
	)

	provider := rules.NewProvider(rulesPgSyncer, rulesLimiter, initRules)
	qp := quota_processor.NewQuotaProcessor(telemetry, provider, config.DefaultTTLDays)

	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		qp.ProcessLogs,
		processorhelper.WithCapabilities(processorCapabilities),
	)
}
