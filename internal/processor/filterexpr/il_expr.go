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

// IlExpr is an interface that evaluates boolean expressions against pdata.InstrumentationLibrary.
type IlExpr interface {
	Evaluate(ils pdata.InstrumentationLibrary) bool
}

type notIl struct {
	a IlExpr
}

func newNotIlExpr(a IlExpr) IlExpr {
	return &notIl{
		a: a,
	}
}

func (na *notIl) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return !na.a.Evaluate(ils)
}

type orIl struct {
	a IlExpr
	b IlExpr
}

func newOrIlExpr(a, b IlExpr) IlExpr {
	return &orIl{
		a: a,
		b: b,
	}
}

func (oa *orIl) Evaluate(ils pdata.InstrumentationLibrary) bool {
	a := oa.a.Evaluate(ils)
	return a || oa.b.Evaluate(ils)
}

type andIl struct {
	a IlExpr
	b IlExpr
}

func newAndIlExpr(a, b IlExpr) IlExpr {
	return &andIl{
		a: a,
		b: b,
	}
}

func (aa *andIl) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return aa.a.Evaluate(ils) && aa.b.Evaluate(ils)
}

type nameIl struct {
	expr StringExpr
}

func newNameIlExpr(expr StringExpr) IlExpr {
	return &nameIl{
		expr: expr,
	}
}

func (nils *nameIl) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return nils.expr.Evaluate(ils.Name())
}

type versionIl struct {
	expr StringExpr
}

func newVersionIlExpr(expr StringExpr) IlExpr {
	return &versionIl{
		expr: expr,
	}
}

func (vils *versionIl) Evaluate(ils pdata.InstrumentationLibrary) bool {
	return vils.expr.Evaluate(ils.Version())
}
