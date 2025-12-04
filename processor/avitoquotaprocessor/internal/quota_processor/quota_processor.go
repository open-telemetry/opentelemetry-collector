package quota_processor

import (
	"context"
	"time"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/metadata"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/rules"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	TTLBucketNameKey = "observer.ttl_bucket_name"
	TTLTimestampKey  = "observer.ttl_timestamp_unix_s"
	PartitionNameKey = "observer.partition_name"
)

type InfraAttributes struct {
	TTLBucketName string    `json:"ttl_bucket_name" ch:"ttl_bucket_name"`
	TTLTimestamp  time.Time `json:"ttl_timestamp" ch:"ttl_timestamp"`
	PartitionName string    `json:"partition_name" ch:"partition_name"`
}

type QuotaProcessor struct {
	rulesFinder    RulesFinder
	defaultTTLDays int

	telemetry *metadata.TelemetryBuilder
}

type RulesFinder interface {
	FindRule(attrs rules.KVGetter) *rules.Rule
}

func NewQuotaProcessor(telemetry *metadata.TelemetryBuilder, rulesFinder RulesFinder, defaultTTLDays int) *QuotaProcessor {
	return &QuotaProcessor{
		rulesFinder:    rulesFinder,
		defaultTTLDays: defaultTTLDays,
		telemetry:      telemetry,
	}
}

func (q *QuotaProcessor) ProcessLogs(ctx context.Context, td plog.Logs) (plog.Logs, error) {
	startBatchProcessing := time.Now()
	logsCount := 0
	defer func() {
		incBatchStats(ctx, q.telemetry, int64(logsCount), time.Since(startBatchProcessing))
	}()

	for i := 0; i < td.ResourceLogs().Len(); i++ {
		resourceLogs := td.ResourceLogs().At(i)
		otelResourceAttributes := resourceLogs.Resource().Attributes()

		system := rules.AnyAttr
		systemRaw, ok := otelResourceAttributes.Get(rules.SystemResourceAttribute)
		if ok {
			system = systemRaw.AsString()
		}

		service := rules.AnyAttr
		serviceRaw, ok := otelResourceAttributes.Get(rules.ServiceResourceAttribute)
		if ok {
			service = serviceRaw.AsString()
		}

		resourceAttributes := otelAttributesToRuleAttributes(otelResourceAttributes)
		for j := 0; j < resourceLogs.ScopeLogs().Len(); j++ {
			scopeLogs := resourceLogs.ScopeLogs().At(j)
			otelScopeAttributes := scopeLogs.Scope().Attributes()
			scopeAttributes := otelAttributesToRuleAttributes(otelScopeAttributes)
			logsCount += scopeLogs.LogRecords().Len()
			for k := 0; k < scopeLogs.LogRecords().Len(); k++ {
				ttl := q.defaultTTLDays
				ruleName := "unknown"
				logRecord := scopeLogs.LogRecords().At(k)
				otelLogAttributes := logRecord.Attributes()
				logAttributes := otelAttributesToRuleAttributes(otelLogAttributes)
				severity := SeverityToString(logRecord.SeverityNumber())
				if severity == "" {
					severity = logRecord.SeverityNumber().String()
				}
				kvGetter := rules.NewAttributes(resourceAttributes, scopeAttributes, logAttributes, system, service, severity)
				rule := q.rulesFinder.FindRule(kvGetter)
				if rule != nil {
					ttl = rule.RetentionDays
					ruleName = rule.Name
				}
				infraAttrs := GetInfraAttrsForRetention(ttl)
				otelLogAttributes.PutStr(PartitionNameKey, infraAttrs.PartitionName)
				otelLogAttributes.PutStr(TTLBucketNameKey, infraAttrs.TTLBucketName)
				otelLogAttributes.PutInt(TTLTimestampKey, infraAttrs.TTLTimestamp.Unix())

				// TODO: optimize allocations here
				incQuotaUsage(ctx, q.telemetry, ruleName, int64(ttl), service, system, severity)
			}
		}
	}
	return td, nil
}

func otelAttributesToRuleAttributes(attrs pcommon.Map) [][2]string {
	ruleAttrs := make([][2]string, 0, attrs.Len())
	attrs.Range(func(k string, v pcommon.Value) bool {
		if k == rules.SystemResourceAttribute || k == rules.ServiceResourceAttribute {
			return true
		}
		switch v.Type() {
		case pcommon.ValueTypeMap:
		case pcommon.ValueTypeSlice:
		case pcommon.ValueTypeBytes:
		case pcommon.ValueTypeEmpty:
			return true
		}
		ruleAttrs = append(ruleAttrs, [2]string{k, v.AsString()})
		return true
	})
	return ruleAttrs
}

func GetInfraAttrsForRetention(retentionDays int) *InfraAttributes {
	t := time.Now()
	ttlTimestamp := t.Add(time.Duration(retentionDays) * 24 * time.Hour)
	ttlBucketName, partitionName := timestampToBucketName(ttlTimestamp, retentionDays)

	return &InfraAttributes{
		TTLBucketName: ttlBucketName,
		TTLTimestamp:  ttlTimestamp,
		PartitionName: partitionName,
	}
}

func timestampToBucketName(t time.Time, retentionDays int) (bucketName, partitionName string) {
	ttlBucketSizeDays := 1
	ttlBucketName := "7d"

	switch duration := retentionDays; {
	case duration > 32:
		ttlBucketSizeDays = 9
		ttlBucketName = "90d"
	case duration > 15:
		ttlBucketSizeDays = 3
		ttlBucketName = "30d"
	case duration > 8:
		ttlBucketSizeDays = 2
		ttlBucketName = "14d"
	default:
		ttlBucketSizeDays = 1
		ttlBucketName = "7d"
	}

	ttlBucketSizeSeconds := ttlBucketSizeDays * 24 * 60 * 60
	ttlBucketTImeUnixSeconds := (t.Unix()/int64(ttlBucketSizeSeconds) + 1) * int64(ttlBucketSizeSeconds)
	ttlBucketTIme := time.Unix(ttlBucketTImeUnixSeconds, 0)
	partitionName = ttlBucketName + "_" + ttlBucketTIme.Format("2006-01-02")

	return ttlBucketName, partitionName
}

const (
	LogSeverityUndefined = "undefined"
	LogSeverityTrace     = "trace"
	LogSeverityDebug     = "debug"
	LogSeverityInfo      = "info"
	LogSeverityWarn      = "warn"
	LogSeverityError     = "error"
	LogSeverityFatal     = "fatal"
)

func SeverityToString(severityNumber plog.SeverityNumber) string {
	switch s := severityNumber; {
	case s > 24:
		return LogSeverityUndefined
	case s > 20:
		return LogSeverityFatal
	case s > 16:
		return LogSeverityError
	case s > 12:
		return LogSeverityWarn
	case s > 8:
		return LogSeverityInfo
	case s > 4:
		return LogSeverityDebug
	case s > 0:
		return LogSeverityTrace
	default:
		return LogSeverityUndefined
	}
}
