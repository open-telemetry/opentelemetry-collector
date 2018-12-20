// Copyright 2018, OpenCensus Authors
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
	"time"

	"github.com/spf13/viper"
)

// SenderType indicates the type of sender
type SenderType string

const (
	// ThriftTChannelSenderType represents a thrift-format tchannel-transport sender
	ThriftTChannelSenderType SenderType = "jaeger-thrift-tchannel"
	// ThriftHTTPSenderType represents a thrift-format http-transport sender
	ThriftHTTPSenderType = "jaeger-thrift-http"
	// InvalidSenderType represents an invalid sender
	InvalidSenderType = "invalid"
)

// JaegerThriftTChannelSenderCfg holds configuration for Jaeger Thrift Tchannel sender
type JaegerThriftTChannelSenderCfg struct {
	CollectorHostPorts        []string      `mapstructure:"collector-host-ports"`
	DiscoveryMinPeers         int           `mapstructure:"discovery-min-peers"`
	DiscoveryConnCheckTimeout time.Duration `mapstructure:"discovery-conn-check-timeout"`
}

// NewJaegerThriftTChannelSenderCfg returns an instance of JaegerThriftTChannelSenderCfg with default values
func NewJaegerThriftTChannelSenderCfg() *JaegerThriftTChannelSenderCfg {
	opts := &JaegerThriftTChannelSenderCfg{
		DiscoveryMinPeers:         3,
		DiscoveryConnCheckTimeout: 250 * time.Millisecond,
	}
	return opts
}

// JaegerThriftHTTPSenderCfg holds configuration for Jaeger Thrift HTTP sender
type JaegerThriftHTTPSenderCfg struct {
	CollectorEndpoint string            `mapstructure:"collector-endpoint"`
	Timeout           time.Duration     `mapstructure:"timeout"`
	Headers           map[string]string `mapstructure:"headers"`
}

// NewJaegerThriftHTTPSenderCfg returns an instance of JaegerThriftHTTPSenderCfg with default values
func NewJaegerThriftHTTPSenderCfg() *JaegerThriftHTTPSenderCfg {
	opts := &JaegerThriftHTTPSenderCfg{
		Timeout: 5 * time.Second,
	}
	return opts
}

// QueuedSpanProcessorCfg holds configuration for the queued span processor
type QueuedSpanProcessorCfg struct {
	// Name is the friendly name of the processor
	Name string
	// NumWorkers is the number of queue workers that dequeue batches and send them out
	NumWorkers int `mapstructure:"num-workers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time
	QueueSize int `mapstructure:"queue-size"`
	// Retry indicates whether queue processor should retry span batches in case of processing failure
	RetryOnFailure bool `mapstructure:"retry-on-failure"`
	// BackoffDelay is the amount of time a worker waits after a failed send before retrying
	BackoffDelay time.Duration `mapstructure:"backoff-delay"`
	// SenderType indicates the type of sender to instantiate
	SenderType   SenderType `mapstructure:"sender-type"`
	SenderConfig interface{}
}

// NewDefaultQueuedSpanProcessorCfg returns an instance of QueuedSpanProcessorCfg with default values
func NewDefaultQueuedSpanProcessorCfg() *QueuedSpanProcessorCfg {
	opts := &QueuedSpanProcessorCfg{
		Name:           "default-queued-jaeger-sender",
		NumWorkers:     10,
		QueueSize:      5000,
		RetryOnFailure: true,
		SenderType:     InvalidSenderType,
		BackoffDelay:   5 * time.Second,
	}
	return opts
}

// InitFromViper initializes QueuedSpanProcessorCfg with properties from viper
func (qOpts *QueuedSpanProcessorCfg) InitFromViper(v *viper.Viper) *QueuedSpanProcessorCfg {
	v.Unmarshal(qOpts)

	switch qOpts.SenderType {
	case ThriftTChannelSenderType:
		ttsopts := NewJaegerThriftTChannelSenderCfg()
		vttsopts := v.Sub(string(ThriftTChannelSenderType))
		if vttsopts != nil {
			vttsopts.Unmarshal(ttsopts)
		}
		qOpts.SenderConfig = ttsopts
	case ThriftHTTPSenderType:
		thsOpts := NewJaegerThriftHTTPSenderCfg()
		vthsOpts := v.Sub(string(ThriftHTTPSenderType))
		if vthsOpts != nil {
			vthsOpts.Unmarshal(thsOpts)
		}
		qOpts.SenderConfig = thsOpts
	}
	return qOpts
}

// MultiSpanProcessorCfg holds configuration for all the span processors
type MultiSpanProcessorCfg struct {
	Processors []*QueuedSpanProcessorCfg
}

// NewDefaultMultiSpanProcessorCfg returns an instance of MultiSpanProcessorCfg with default values
func NewDefaultMultiSpanProcessorCfg() *MultiSpanProcessorCfg {
	opts := &MultiSpanProcessorCfg{
		Processors: make([]*QueuedSpanProcessorCfg, 0),
	}
	return opts
}

// InitFromViper initializes MultiSpanProcessorCfg with properties from viper
func (mOpts *MultiSpanProcessorCfg) InitFromViper(v *viper.Viper) *MultiSpanProcessorCfg {
	const baseKey = "queued-exporters"
	procsv := v.Sub(baseKey)
	if procsv == nil {
		return mOpts
	}
	for procName := range v.GetStringMap(baseKey) {
		procv := procsv.Sub(procName)
		procOpts := NewDefaultQueuedSpanProcessorCfg()
		procOpts.Name = procName
		procOpts.InitFromViper(procv)
		mOpts.Processors = append(mOpts.Processors, procOpts)
	}
	return mOpts
}
