// Copyright The OpenTelemetry Authors
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

package labelfilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type labelFilterProcessor struct {
	cfg *Config
}

var _ processorhelper.MProcessor = (*labelFilterProcessor)(nil)

func newLabelFilterProcessor(cfg *Config) (*labelFilterProcessor, error) {
	return &labelFilterProcessor{cfg: cfg}, nil
}

func (p *labelFilterProcessor) ProcessMetrics(ctx context.Context, metrics pdata.Metrics) (pdata.Metrics, error) {
	return pdata.Metrics{}, nil
}
