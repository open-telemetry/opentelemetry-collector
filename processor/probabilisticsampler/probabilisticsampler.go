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

package probabilisticsampler

import (
	"context"
	"errors"
	"fmt"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

const (
	// The constants below are tags used to read the configuration via viper.
	samplingPercentageCfgTag = "sampling-percentage"
	hashSeedCfgTag           = "hash-seed"

	// The constants help translate user friendly percentages to numbers direct used in sampling.
	numHashBuckets        = 0x4000 // Using a power of 2 to avoid division.
	bitMaskHashBuckets    = numHashBuckets - 1
	percentageScaleFactor = numHashBuckets / 100.0
)

// InitFromViper updates TraceSampler config according to the viper configuration.
func (tsc *Config) InitFromViper(v *viper.Viper) (*Config, error) {
	if v == nil {
		return nil, errors.New("v is nil")
	}
	if err := v.UnmarshalKey(samplingPercentageCfgTag, &tsc.SamplingPercentage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q: %v", samplingPercentageCfgTag, err)
	}
	if err := v.UnmarshalKey(hashSeedCfgTag, &tsc.HashSeed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q: %v", hashSeedCfgTag, err)
	}
	return tsc, nil
}

type tracesamplerprocessor struct {
	nextConsumer       consumer.TraceConsumer
	scaledSamplingRate uint32
	hashSeed           uint32
}

var _ processor.TraceProcessor = (*tracesamplerprocessor)(nil)

// NewTraceProcessor returns a processor.TraceProcessor that will perform head sampling according to the given
// configuration.
func NewTraceProcessor(nextConsumer consumer.TraceConsumer, cfg Config) (processor.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, errors.New("nextConsumer is nil")
	}

	return &tracesamplerprocessor{
		nextConsumer: nextConsumer,
		// Adjust sampling percentage on private so recalculations are avoided.
		scaledSamplingRate: uint32(cfg.SamplingPercentage * percentageScaleFactor),
		hashSeed:           cfg.HashSeed,
	}, nil
}

func (tsp *tracesamplerprocessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	scaledSamplingRate := tsp.scaledSamplingRate
	if scaledSamplingRate >= numHashBuckets {
		return tsp.nextConsumer.ConsumeTraceData(ctx, td)
	}

	sampledTraceData := consumerdata.TraceData{
		Node:         td.Node,
		Resource:     td.Resource,
		SourceFormat: td.SourceFormat,
	}

	sampledSpans := make([]*tracepb.Span, 0, len(td.Spans))
	for _, span := range td.Spans {
		// If one assumes random trace ids hashing may seems avoidable, however, traces can be coming from sources
		// with various different criteria to generate trace id and perhaps were already sampled without hashing.
		// Hashing here prevents bias due to such systems.
		if hash(span.TraceId, tsp.hashSeed)&bitMaskHashBuckets < scaledSamplingRate {
			sampledSpans = append(sampledSpans, span)
		}
	}

	sampledTraceData.Spans = sampledSpans

	return tsp.nextConsumer.ConsumeTraceData(ctx, sampledTraceData)
}

// hash is a murmur3 hash function, see http://en.wikipedia.org/wiki/MurmurHash.
func hash(key []byte, seed uint32) (hash uint32) {
	const (
		c1 = 0xcc9e2d51
		c2 = 0x1b873593
		c3 = 0x85ebca6b
		c4 = 0xc2b2ae35
		r1 = 15
		r2 = 13
		m  = 5
		n  = 0xe6546b64
	)

	hash = seed
	iByte := 0
	for ; iByte+4 <= len(key); iByte += 4 {
		k := uint32(key[iByte]) | uint32(key[iByte+1])<<8 | uint32(key[iByte+2])<<16 | uint32(key[iByte+3])<<24
		k *= c1
		k = (k << r1) | (k >> (32 - r1))
		k *= c2
		hash ^= k
		hash = (hash << r2) | (hash >> (32 - r2))
		hash = hash*m + n
	}

	// TraceId and SpanId have lengths that are multiple of 4 so the code below is never expected to
	// be hit when sampling traces. However, it is preserved here to keep it as a correct murmur3 implementation.
	// This is enforced via tests.
	var remainingBytes uint32
	switch len(key) - iByte {
	case 3:
		remainingBytes += uint32(key[iByte+2]) << 16
		fallthrough
	case 2:
		remainingBytes += uint32(key[iByte+1]) << 8
		fallthrough
	case 1:
		remainingBytes += uint32(key[iByte])
		remainingBytes *= c1
		remainingBytes = (remainingBytes << r1) | (remainingBytes >> (32 - r1))
		remainingBytes = remainingBytes * c2
		hash ^= remainingBytes
	}

	hash ^= uint32(len(key))
	hash ^= hash >> 16
	hash *= c3
	hash ^= hash >> 13
	hash *= c4
	hash ^= hash >> 16

	return
}
