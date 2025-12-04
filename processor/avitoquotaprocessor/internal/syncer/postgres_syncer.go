package syncer

import (
	"context"
	"sync"
	"time"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/generated/rpc/clients/paas_observability"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/pkg/pointers"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/rules"
	"go.uber.org/zap"
)

const EVENT_RATE = "logsPerSec"
const STORAGE_SIZE = "logsStorage"

type observabilityClient interface {
	ListLogsRules(ctx context.Context, in *paas_observability.ListLogsRulesIn) (*paas_observability.ListLogsRulesOut, error)
}

type postgresSync struct {
	observabilityClient observabilityClient
	clientID            string

	rulesWatchersMu sync.RWMutex
	rulesWatchers   []chan rules.Rules
	lastRules       rules.Rules

	rulesUpdateInterval time.Duration

	telemetry *metadata.TelemetryBuilder
	logger    *zap.Logger
}

func NewPostgresSyncLimiter(logger *zap.Logger, telemetry *metadata.TelemetryBuilder, observabilityClient observabilityClient, clientID string, rulesUpdateInterval time.Duration) *postgresSync {
	return &postgresSync{
		observabilityClient: observabilityClient,
		clientID:            clientID,
		rulesUpdateInterval: rulesUpdateInterval,
		telemetry:           telemetry,
		logger:              logger.Named("postgres_sync").With(zap.String("client_id", clientID), zap.Duration("rules_update_interval", rulesUpdateInterval)),
	}
}

func (r *postgresSync) Start(ctx context.Context) {
	go r.watchRulesLoop(ctx)
}

func (r *postgresSync) WatchRulesUpdates() <-chan rules.Rules {
	ch := make(chan rules.Rules)
	r.rulesWatchersMu.Lock()
	defer r.rulesWatchersMu.Unlock()
	r.logger.Debug("new rules watcher")
	r.rulesWatchers = append(r.rulesWatchers, ch)
	return ch
}

func (r *postgresSync) watchRulesLoop(ctx context.Context) {
	ticker := time.NewTicker(r.rulesUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.updateRules(ctx)
		}
	}
}

func (r *postgresSync) GetRules(ctx context.Context) (rules.Rules, error) {

	rawRulesFromObservabilityClient, err := r.observabilityClient.ListLogsRules(ctx, &paas_observability.ListLogsRulesIn{})
	if err != nil {
		return nil, err
	}
	rules := rawRulesFromObservabilityClientToRules(rawRulesFromObservabilityClient)
	r.lastRules = rules
	return rules, nil
}

func rawRulesFromObservabilityClientToRules(out *paas_observability.ListLogsRulesOut) (rules rules.Rules) {
	for _, rule := range out.Rules {
		rules = append(rules, rawObservabilityRuleToRule(rule))
	}
	return
}

func rawObservabilityRuleToRule(in paas_observability.ActualLogsRule) *rules.Rule {
	var (
		ratePerSecond int
		storageSize   int
	)
	for _, quota := range in.Quotas {
		switch quota.ResourceMetricID {
		case EVENT_RATE:
			ratePerSecond = int(quota.Value)
		case STORAGE_SIZE:
			storageSize = int(quota.Value)
		}
	}
	retensionDays := int(in.Ttl.DurationSeconds / (60 * 60 * 24))
	return &rules.Rule{
		ID:                in.RuleID,
		Name:              in.RuleName,
		Expressions:       ruleExpressionsFromFilter(in.Filter),
		RatePerSecond:     ratePerSecond,
		StorageRatePerDay: storageSize / retensionDays,
		RetentionDays:     retensionDays,
	}

}

func ruleExpressionsFromFilter(filter []paas_observability.LogsAttributeExpression) (expressions []rules.Expression) {
	for _, filter := range filter {
		expressions = append(expressions, rules.Expression{
			Key:       filter.Key.Name,
			Operation: pointers.ToString(filter.Operator, "="),
			Value:     pointers.ToString(filter.Value, ""),
		})
	}
	return
}

func isRulesChanged(oldRules, newRules rules.Rules) bool {
	if len(oldRules) != len(newRules) {
		return true
	}
	oldRulesMap := make(map[string]*rules.Rule, len(oldRules))
	for i, oldRule := range oldRules {
		oldRulesMap[oldRule.ID] = oldRules[i]
	}
	for _, newRule := range newRules {
		oldRule, ok := oldRulesMap[newRule.ID]
		if !ok {
			return true
		}
		if !oldRule.IsEqual(newRule) {
			return true
		}
	}
	return false
}

func (r *postgresSync) updateRules(ctx context.Context) {
	lastRules := r.lastRules
	newRules, err := r.GetRules(ctx)
	if err != nil {
		r.logger.Error("failed to get rules", zap.Error(err))
		incSyncRules(ctx, r.telemetry, "failed")
		return
	}
	if !isRulesChanged(lastRules, newRules) {
		incSyncRules(ctx, r.telemetry, "unchanged")
		return
	}
	r.rulesWatchersMu.Lock()
	defer r.rulesWatchersMu.Unlock()
	for _, ch := range r.rulesWatchers {
		select {
		case ch <- newRules:
		default:
		}
	}

	r.logger.Debug("rule updated count", zap.Int("count", len(newRules)))

	incSyncRules(ctx, r.telemetry, "updated")
}
