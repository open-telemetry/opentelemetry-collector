// Copyright 2020 OpenTelemetry Authors
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

package receivertest

import (
	"context"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

type TraceAppender struct {
	sync.RWMutex
	TraceDataList []consumerdata.TraceData
}

func NewTraceAppender() *TraceAppender {
	return &TraceAppender{}
}

var _ consumer.TraceConsumer = (*TraceAppender)(nil)

func (ma *TraceAppender) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ma.Lock()
	defer ma.Unlock()

	ma.TraceDataList = append(ma.TraceDataList, td)

	return nil
}
