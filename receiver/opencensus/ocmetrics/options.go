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

package ocmetrics

import "time"

// Option interface defines for configuration settings to be applied to receivers.
//
// WithReceiver applies the configuration to the given receiver.
type Option interface {
	WithReceiver(*Receiver)
}

type metricBufferPeriod struct {
	period time.Duration
}

var _ Option = (*metricBufferPeriod)(nil)

func (mfd *metricBufferPeriod) WithReceiver(ocr *Receiver) {
	ocr.metricBufferPeriod = mfd.period
}

// WithMetricBufferPeriod is an option that allows one to configure
// the period that spans are buffered for before the Receiver
// sends them to its MetricsReceiver.
func WithMetricBufferPeriod(period time.Duration) Option {
	return &metricBufferPeriod{period: period}
}

type metricBufferCount int

var _ Option = (*metricBufferCount)(nil)

func (mpc metricBufferCount) WithReceiver(oci *Receiver) {
	oci.metricBufferCount = int(mpc)
}

// WithMetricBufferCount is an option that allows one to configure
// the number of metrics that are buffered before the Receiver
// send them to its MetricsReceiverSink.
func WithMetricBufferCount(count int) Option {
	return metricBufferCount(count)
}
