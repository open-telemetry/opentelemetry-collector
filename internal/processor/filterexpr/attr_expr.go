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

// AttributeExpr is an interface that evaluates boolean expressions against pdata.AttributeMap.
type AttributeExpr interface {
	Evaluate(attributes pdata.AttributeMap) bool
}

type hasAttribute struct {
	key string
}

func newHasAttributeExpr(key string) AttributeExpr {
	return &hasAttribute{
		key: key,
	}
}

func (ha *hasAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	_, ok := attributes.Get(ha.key)
	return ok
}

type doubleAttribute struct {
	key  string
	expr DoubleExpr
}

func newDoubleAttributeExpr(key string, expr DoubleExpr) AttributeExpr {
	return &doubleAttribute{
		key:  key,
		expr: expr,
	}
}

func (da *doubleAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(da.key)
	return ok && da.expr.Evaluate(val.DoubleVal())
}

type intAttribute struct {
	key  string
	expr IntExpr
}

func newIntAttributeExpr(key string, expr IntExpr) AttributeExpr {
	return &intAttribute{
		key:  key,
		expr: expr,
	}
}

func (ia *intAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(ia.key)
	return ok && ia.expr.Evaluate(val.IntVal())
}

type boolAttribute struct {
	key string
	val bool
}

func newBoolAttributeExpr(key string, val bool) AttributeExpr {
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

func newStringAttributeExpr(key string, expr StringExpr) AttributeExpr {
	return &stringAttribute{
		key:  key,
		expr: expr,
	}
}

func (sa *stringAttribute) Evaluate(attributes pdata.AttributeMap) bool {
	val, ok := attributes.Get(sa.key)
	return ok && sa.expr.Evaluate(val.StringVal())
}
