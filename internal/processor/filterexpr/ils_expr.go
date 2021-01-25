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

package filterexpr

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// IlsExpr is an interface that evaluates boolean expressions for pdata.InstrumentationLibrary.
type IlsExpr interface {
	Evaluate(ils pdata.InstrumentationLibrary) bool
}

type notIls struct {
	a IlsExpr
}

func NewNotIlsExpr(a IlsExpr) IlsExpr {
	return &notIls{
		a: a,
	}
}

func (na *notIls) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return !na.a.Evaluate(ils)
}

type orIls struct {
	a IlsExpr
	b IlsExpr
}

func NewOrIlsExpr(a, b IlsExpr) IlsExpr {
	return &orIls{
		a: a,
		b: b,
	}
}

func (oa *orIls) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return oa.a.Evaluate(ils) || oa.b.Evaluate(ils)
}

type andIls struct {
	a IlsExpr
	b IlsExpr
}

func NewAndIlsExpr(a, b IlsExpr) IlsExpr {
	return &andIls{
		a: a,
		b: b,
	}
}

func (aa *andIls) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return aa.a.Evaluate(ils) && aa.b.Evaluate(ils)
}

type nameIls struct {
	expr StringExpr
}

func NewNameIlsExpr(expr StringExpr) IlsExpr {
	return &nameIls{
		expr: expr,
	}
}

func (nils *nameIls) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return nils.expr.Evaluate(ils.Name())
}

type versionIls struct {
	expr StringExpr
}

func NewVersionIlsExpr(expr StringExpr) IlsExpr {
	return &versionIls{
		expr: expr,
	}
}

func (vils *versionIls) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return vils.expr.Evaluate(ils.Version())
}
