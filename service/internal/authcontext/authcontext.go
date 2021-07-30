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

package authcontext

import (
	"context"

	"go.opentelemetry.io/collector/config/configauth"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

func NewTracesInjector(next consumer.Traces) consumer.Traces {
	return injectorForTraces{next}
}

type injectorForTraces struct {
	next consumer.Traces
}

func (tc injectorForTraces) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (tc injectorForTraces) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	authContext, ok := configauth.ExtractAuthContext(ctx)
	if ok && authContext != nil {
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			rss := td.ResourceSpans().At(i)
			configauth.InjectPDataContext(rss.PDataContext(), authContext)
		}
	}

	return tc.next.ConsumeTraces(ctx, td)
}
