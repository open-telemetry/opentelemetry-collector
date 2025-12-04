package syncer

import (
	"reflect"
	"testing"

	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/generated/rpc/clients/paas_observability"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/pkg/pointers"
	"go.avito.ru/arch/opentelemetry-collector-avito/processor/avitoquotaprocessor/internal/rules"
)

func Test_rawObservabilityRuleToRule(t *testing.T) {
	type args struct {
		in paas_observability.ActualLogsRule
	}
	tests := []struct {
		name string
		args args
		want *rules.Rule
	}{
		{
			name: "rule-1",
			args: args{
				in: paas_observability.ActualLogsRule{
					RuleID: "6c6e0168-930c-4ee1-b10a-f93c521316b7",
					Filter: []paas_observability.LogsAttributeExpression{
						{
							Key: paas_observability.LogsAttributeKeyV3{
								Name: "service",
								Kind: "system",
							},
							Operator: pointers.FromString("="),
							Value:    pointers.FromString("iam"),
							Values:   nil,
						},
						{
							Key: paas_observability.LogsAttributeKeyV3{
								Name: "service2",
								Kind: "system",
							},
							Operator: nil,
							Value:    pointers.FromString("iam2"),
							Values:   nil,
						},
						{
							Key: paas_observability.LogsAttributeKeyV3{
								Name: "service3",
								Kind: "system",
							},
							Operator: pointers.FromString(">"),
							Value:    pointers.FromString("iam3"),
							Values:   nil,
						},
					},
					Quotas: []paas_observability.LogsRuleVersionQuota{
						{
							ResourceMetricID: "logsPerSec",
							Value:            100,
						},
						{
							ResourceMetricID: "someOtherResource",
							Value:            200,
						},
						{
							ResourceMetricID: "logsStorage",
							Value:            1000,
						},
					},
					Ttl: paas_observability.LogsRuleTTLProfile{
						Id:              "14d",
						ResourceId:      "ch-opentracing",
						Name:            "14d",
						DurationSeconds: 15 * 24 * 60 * 60,
						IsDefault:       false,
					},
				},
			},
			want: &rules.Rule{
				ID:   "6c6e0168-930c-4ee1-b10a-f93c521316b7",
				Name: "6c6e0168-930c-4ee1-b10a-f93c521316b7",
				Expressions: []rules.Expression{
					{
						Key:       "service",
						Operation: "=",
						Value:     "iam",
					},
					{
						Key:       "service2",
						Operation: "=",
						Value:     "iam2",
					},
					{
						Key:       "service3",
						Operation: ">",
						Value:     "iam3",
					},
				},
				RatePerSecond:     100,
				StorageRatePerDay: 1000,
				RetentionDays:     15,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rawObservabilityRuleToRule(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rawObservabilityRuleToRule() = %v, want %v", got, tt.want)
			}
		})
	}
}
