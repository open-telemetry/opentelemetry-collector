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

package ocinterceptor

import "time"

type OCOption interface {
	WithOCInterceptor(*OCInterceptor)
}

type spanBufferPeriod struct {
	period time.Duration
}

var _ OCOption = (*spanBufferPeriod)(nil)

func (sfd *spanBufferPeriod) WithOCInterceptor(oci *OCInterceptor) {
	oci.spanBufferPeriod = sfd.period
}

// WithSpanBufferPeriod is an option that allows one to configure
// the period that spans are buffered for before the OCInterceptor
// sends them to its SpanReceiver.
func WithSpanBufferPeriod(period time.Duration) OCOption {
	return &spanBufferPeriod{period: period}
}
