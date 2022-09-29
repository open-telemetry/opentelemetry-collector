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

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import (
	"sync"

	otelview "go.opentelemetry.io/otel/sdk/metric/view"
)

type views struct {
	mtx sync.Mutex
	v   []otelview.View
}

func (vs *views) addViews(v ...otelview.View) {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()
	vs.v = append(vs.v, v...)
}

func (vs *views) getViews() []otelview.View {
	vs.mtx.Lock()
	defer vs.mtx.Unlock()
	return storage.v
}

var storage = views{
	v: []otelview.View{},
}

// AddViews to the OpenTelemetry Go SDK.
// Views are only configurable at the SDK initialization, for this reason
// components must call this function inside an init() function to be correctly initialized.
func AddViews(v ...otelview.View) {
	storage.addViews(v...)
}

// GetViews returns the registered views
func GetViews() []otelview.View {
	return storage.getViews()
}
