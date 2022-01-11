// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config // import "go.opentelemetry.io/collector/config"

import (
	"context"
)

// MapConverter interface to convert a config.Map. This allows distributions
// (or components) to build backwards compatible config converters.
type MapConverter interface {
	Convert(ctx context.Context, cfgMap *Map) error

	// privateMapConverter is an unexported func to disallow direct implementation.
	privateMapConverter()
}

// ConvertFunc specifies the function invoked when the MapConverter.Convert is being called.
type ConvertFunc func(context.Context, *Map) error

// Convert implements the MapConverter.Convert.
func (f ConvertFunc) Convert(ctx context.Context, cfgMap *Map) error {
	if f == nil {
		return nil
	}
	return f(ctx, cfgMap)
}

type converter struct {
	ConvertFunc
}

func (converter) privateMapConverter() {}

// NewMapConverter returns a MapConverter configured with the provided options.
func NewMapConverter(convertFunc ConvertFunc) MapConverter {
	return &converter{
		ConvertFunc: convertFunc,
	}
}
