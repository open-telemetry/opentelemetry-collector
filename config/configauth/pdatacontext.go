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

package configauth

import (
	"context"

	"go.opentelemetry.io/collector/model/pdata"
)

type ctxKey struct{}

type AuthContext struct {
	sub   string
	raw   string
	group []string
}

func InjectAuthContext(ctx context.Context, sub, raw string, group []string) context.Context {
	return context.WithValue(ctx, ctxKey{}, &AuthContext{
		sub:   sub,
		raw:   raw,
		group: group,
	})
}

func ExtractAuthContext(ctx context.Context) (*AuthContext, bool) {
	ac, ok := ctx.Value(ctxKey{}).(*AuthContext)
	if !ok {
		return nil, false
	}
	return ac, true
}

func InjectPDataContext(pda pdata.PDataContext, ac *AuthContext) {
	pda.Set(ctxKey{}, ac)
}

func ExtractPDataContext(pda pdata.PDataContext) *AuthContext {
	return pda.Get(ctxKey{}).(*AuthContext)
}

func (ac *AuthContext) Subject() string {
	return ac.sub
}
func (ac *AuthContext) Raw() string {
	return ac.raw
}
func (ac *AuthContext) Groups() []string {
	return ac.group
}
