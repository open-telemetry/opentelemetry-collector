// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tailsamplingprocessor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config/configmodels"
)

func TestCompositeHelper(t *testing.T) {
	cfg := &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: "tail_sampling",
			NameVal: "tail_sampling",
		},
		DecisionWait:            10 * time.Second,
		NumTraces:               100,
		ExpectedNewTracesPerSec: 10,
		PolicyCfgs: []PolicyCfg{
			{
				Name: "composite-policy-1",
				Type: Composite,
				CompositeCfg: CompositeCfg{
					MaxTotalSpansPerSecond: 1000,
					PolicyOrder:            []string{"test-composite-policy-1", "test-composite-policy-2", "test-composite-policy-3", "test-composite-policy-4", "test-composite-policy-5"},
					SubPolicyCfg: []SubPolicyCfg{
						{
							Name:                "test-composite-policy-1",
							Type:                NumericAttribute,
							NumericAttributeCfg: NumericAttributeCfg{Key: "key1", MinValue: 50, MaxValue: 100},
						},
						{
							Name:               "test-composite-policy-2",
							Type:               StringAttribute,
							StringAttributeCfg: StringAttributeCfg{Key: "key2", Values: []string{"value1", "value2"}},
						},
						{
							Name:            "test-composite-policy-3",
							Type:            RateLimiting,
							RateLimitingCfg: RateLimitingCfg{SpansPerSecond: 10},
						},
						{
							Name: "test-composite-policy-4",
							Type: AlwaysSample,
						},
						{
							Name: "test-composite-policy-5",
						},
					},
					RateAllocation: []RateAllocationCfg{
						{
							Policy:  "test-composite-policy-1",
							Percent: 50,
						},
						{
							Policy:  "test-composite-policy-2",
							Percent: 25,
						},
					},
				},
			},
		},
	}
	rlfCfg := cfg.PolicyCfgs[0].CompositeCfg
	composite, e := getNewCompositePolicy(zap.NewNop(), rlfCfg)
	require.NotNil(t, composite)
	require.Nil(t, e)
}
