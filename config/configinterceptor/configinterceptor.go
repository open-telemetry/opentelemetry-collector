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

package configinterceptor // import "go.opentelemetry.io/collector/config/configinterceptor"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/interceptor"
)

type Interceptors struct {
	Interceptors []component.ID `mapstructure:"-"`
}

func (i Interceptors) GetInterceptors(extensions map[component.ID]component.Component) ([]interceptor.Interceptor, error) {
	ret := make([]interceptor.Interceptor, 0)
	for _, interceptorID := range i.Interceptors {
		if ext, found := extensions[interceptorID]; found {
			if i, ok := ext.(interceptor.Interceptor); ok {
				ret = append(ret, i)
			} else {
				return nil, fmt.Errorf("the component %q is not an interceptor", interceptorID)
			}
		} else {
			return nil, fmt.Errorf("the requested interceptor %q was not found", interceptorID)
		}
	}
	return ret, nil
}
