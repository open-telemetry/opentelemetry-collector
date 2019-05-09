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

	"github.com/census-instrumentation/opencensus-service/processor/attributekeyprocessor"
)

// SenderType indicates the type of sender
type SenderType string

const (
	// ThriftTChannelSenderType represents a thrift-format tchannel-transport sender
	ThriftTChannelSenderType SenderType = "jaeger-thrift-tchannel"
	// ThriftHTTPSenderType represents a thrift-format http-transport sender
	ThriftHTTPSenderType = "jaeger-thrift-http"
	// ProtoGRPCSenderType represents a proto-format grpc-transport sender
	ProtoGRPCSenderType = "jaeger-proto-grpc"
	// InvalidSenderType represents an invalid sender
	InvalidSenderType = "invalid"
)

const (
	queuedExportersConfigKey = "queued-exporters"
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

// JaegerProtoGRPCSenderCfg holds configuration for Jaeger Proto GRPC sender
type JaegerProtoGRPCSenderCfg struct {
	CollectorEndpoint string `mapstructure:"collector-endpoint"`
}

// NewJaegerProtoGRPCSenderCfg returns an instance of JaegerProtoGRPCSenderCfg with default values
func NewJaegerProtoGRPCSenderCfg() *JaegerProtoGRPCSenderCfg {
	return &JaegerProtoGRPCSenderCfg{}
}

// BatchingConfig contains configuration around the queueing batching.
// It contains some advanced configurations, which should not be used
// by a typical user, but are provided as advanced features to increase
// scalability.
type BatchingConfig struct {
	// Enable marks batching as enabled or not
	Enable bool `mapstructure:"enable"`
	// Timeout sets the time after which a batch will be sent regardless of size
	Timeout *time.Duration `mapstructure:"timeout,omitempty"`
	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	SendBatchSize *int `mapstructure:"send-batch-size,omitempty"`

	// NumTickers sets the number of tickers to use to divide the work of looping
	// over batch buckets. This is an advanced configuration option.
	NumTickers int `mapstructure:"num-tickers,omitempty"`
	// TickTime sets time interval at which the tickers tick. This is an advanced
	// configuration option.
	TickTime *time.Duration `mapstructure:"tick-time,omitempty"`
	// RemoveAfterTicks is the number of ticks that must pass without a span arriving
	// from a node after which the batcher for that node will be deleted. This is an
	// advanved configuration option.
	RemoveAfterTicks *int `mapstructure:"remove-after-ticks,omitempty"`
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
	// BatchingConfig sets config parameters related to batching
	BatchingConfig BatchingConfig `mapstructure:"batching"`
	RawConfig      *viper.Viper
}

// AttributesCfg holds configuration for attributes that can be added to all spans
// going through a processor.
type AttributesCfg struct {
	Overwrite       bool                                   `mapstructure:"overwrite"`
	Values          map[string]interface{}                 `mapstructure:"values"`
	KeyReplacements []attributekeyprocessor.KeyReplacement `mapstructure:"key-mapping,omitempty"`
}

// GlobalProcessorCfg holds global configuration values that apply to all processors
type GlobalProcessorCfg struct {
	Attributes *AttributesCfg `mapstructure:"attributes"`
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
	case ProtoGRPCSenderType:
		pgopts := NewJaegerProtoGRPCSenderCfg()
		vpgopts := v.Sub(string(ProtoGRPCSenderType))
		if vpgopts != nil {
			vpgopts.Unmarshal(pgopts)
		}
		qOpts.SenderConfig = pgopts
	}
	qOpts.RawConfig = v
	return qOpts
}

// MultiSpanProcessorCfg holds configuration for all the span processors
type MultiSpanProcessorCfg struct {
	Processors []*QueuedSpanProcessorCfg
	Global     *GlobalProcessorCfg `mapstructure:"global"`
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
	procsv := v.Sub(queuedExportersConfigKey)
	if procsv != nil {
		for procName := range v.GetStringMap(queuedExportersConfigKey) {
			procv := procsv.Sub(procName)
			procOpts := NewDefaultQueuedSpanProcessorCfg()
			procOpts.Name = procName
			procOpts.InitFromViper(procv)
			mOpts.Processors = append(mOpts.Processors, procOpts)
		}
	}

	vglobal := v.Sub("global")
	if vglobal != nil {
		global := &GlobalProcessorCfg{}
		err := vglobal.Unmarshal(global)
		if err == nil {
			mOpts.Global = global
		}
	}

	return mOpts
}
