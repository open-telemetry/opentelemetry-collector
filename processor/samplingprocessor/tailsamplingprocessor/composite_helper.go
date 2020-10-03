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
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/processor/samplingprocessor/tailsamplingprocessor/sampling"
)

func getNewCompositePolicy(logger *zap.Logger, config CompositeCfg) (sampling.PolicyEvaluator, error) {
	var subPolicyEvalParams []sampling.SubPolicyEvalParams
	rateAllocationsMap := getRateAllocationMap(config)
	for i := range config.SubPolicyCfg {
		policyCfg := &config.SubPolicyCfg[i]
		policy, _ := getSubPolicyEvaluator(logger, policyCfg)

		evalParams := sampling.SubPolicyEvalParams{
			Evaluator:         policy,
			MaxSpansPerSecond: int64(rateAllocationsMap[policyCfg.Name]),
		}
		subPolicyEvalParams = append(subPolicyEvalParams, evalParams)
	}
	return sampling.NewComposite(logger, config.MaxTotalSpansPerSecond, subPolicyEvalParams, sampling.MonotonicClock{}), nil
}

// Apply rate allocations to the sub-policies
func getRateAllocationMap(config CompositeCfg) map[string]float64 {
	rateAllocationsMap := make(map[string]float64)
	maxTotalSPS := float64(config.MaxTotalSpansPerSecond)
	// Default SPS determined by equally diving number of sub policies
	defaultSPS := maxTotalSPS / float64(len(config.SubPolicyCfg))
	for i := 0; i < len(config.RateAllocation); i++ {
		rAlloc := &config.RateAllocation[i]
		rateAllocationsMap[rAlloc.Policy] = defaultSPS
		if rAlloc.Percent > 0 {
			rateAllocationsMap[rAlloc.Policy] = (float64(rAlloc.Percent) / 100) * maxTotalSPS
		}
	}
	return rateAllocationsMap
}

// Return instance of composite sub-policy
func getSubPolicyEvaluator(logger *zap.Logger, cfg *SubPolicyCfg) (sampling.PolicyEvaluator, error) {
	switch cfg.Type {
	case AlwaysSample:
		return sampling.NewAlwaysSample(logger), nil
	case NumericAttribute:
		nafCfg := cfg.NumericAttributeCfg
		return sampling.NewNumericAttributeFilter(logger, nafCfg.Key, nafCfg.MinValue, nafCfg.MaxValue), nil
	case StringAttribute:
		safCfg := cfg.StringAttributeCfg
		return sampling.NewStringAttributeFilter(logger, safCfg.Key, safCfg.Values), nil
	case RateLimiting:
		rlfCfg := cfg.RateLimitingCfg
		return sampling.NewRateLimiting(logger, rlfCfg.SpansPerSecond), nil
	default:
		return nil, fmt.Errorf("unknown sampling policy type %s", cfg.Type)
	}
}
