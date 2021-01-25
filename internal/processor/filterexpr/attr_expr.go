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

// AttributeExpr is an interface that evaluates boolean expressions for pdata.AttributeMap.
type AttributeExpr interface {
	Evaluate(attributes pdata.AttributeMap) bool
}

type notAttribute struct {
	a AttributeExpr
}

func NewNotAttributeExpr(a AttributeExpr) AttributeExpr {
	return &notAttribute{
		a: a,
	}
}

func (na *notAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	return !na.a.Evaluate(attributes)
}

type orAttribute struct {
	a AttributeExpr
	b AttributeExpr
}

func NewOrAttributeExpr(a, b AttributeExpr) AttributeExpr {
	return &orAttribute{
		a: a,
		b: b,
	}
}

func (oa *orAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	return oa.a.Evaluate(attributes) || oa.b.Evaluate(attributes)
}

type andAttribute struct {
	a AttributeExpr
	b AttributeExpr
}

func NewAndAttributeExpr(a, b AttributeExpr) AttributeExpr {
	return &andAttribute{
		a: a,
		b: b,
	}
}

func (aa *andAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	return aa.a.Evaluate(attributes) && aa.b.Evaluate(attributes)
}

type hasAttribute struct {
	key string
}

func NewHasAttributeExpr(key string) AttributeExpr {
	return &hasAttribute{
		key: key,
	}
}

func (ha *hasAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	_, ok := attributes.Get(ha.key)
	return ok
}

type doubleAttribute struct {
	key string
	val float64
}

func NewDoubleAttributeExpr(key string, val float64) AttributeExpr {
	return &doubleAttribute{
		key: key,
		val: val,
	}
}

func (da *doubleAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(da.key)
	return ok && val.DoubleVal() == da.val
}

type intAttribute struct {
	key string
	val int64
}

func NewIntAttributeExpr(key string, val int64) AttributeExpr {
	return &intAttribute{
		key: key,
		val: val,
	}
}

func (ia *intAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(ia.key)
	return ok && val.IntVal() == ia.val
}

type boolAttribute struct {
	key string
	val bool
}

func NewBoolAttributeExpr(key string, val bool) AttributeExpr {
	return &boolAttribute{
		key: key,
		val: val,
	}
}

func (ba *boolAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(ba.key)
	return ok && val.BoolVal() == ba.val
}

type stringAttribute struct {
	key  string
	expr StringExpr
}

func NewStringAttributeExpr(key string, expr StringExpr) AttributeExpr {
	return &stringAttribute{
		key:  key,
		expr: expr,
	}
}

func (sa *stringAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(sa.key)
	return ok && sa.expr.Evaluate(val.StringVal())
}
