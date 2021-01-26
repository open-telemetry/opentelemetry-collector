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

// ResourceExpr is an interface that evaluates boolean expressions against pdata.Resource.
type ResourceExpr interface {
	Evaluate(resource pdata.Resource) bool
}

type notResource struct {
	a ResourceExpr
}

func newNotResourceExpr(a ResourceExpr) ResourceExpr {
	return &notResource{
		a: a,
	}
}

func (na *notResource) Evaluate(resource pdata.Resource) bool {
	return !na.a.Evaluate(resource)
}

type orResource struct {
	a ResourceExpr
	b ResourceExpr
}

func newOrResourceExpr(a, b ResourceExpr) ResourceExpr {
	return &orResource{
		a: a,
		b: b,
	}
}

func (oa *orResource) Evaluate(resource pdata.Resource) bool {
	return oa.a.Evaluate(resource) || oa.b.Evaluate(resource)
}

type andResource struct {
	a ResourceExpr
	b ResourceExpr
}

func newAndResourceExpr(a, b ResourceExpr) ResourceExpr {
	return &andResource{
		a: a,
		b: b,
	}
}

func (aa *andResource) Evaluate(resource pdata.Resource) bool {
	return aa.a.Evaluate(resource) && aa.b.Evaluate(resource)
}

type attributeResource struct {
	expr AttributeExpr
}

func newAttributesResourceExpr(expr AttributeExpr) ResourceExpr {
	return &attributeResource{
		expr: expr,
	}
}

func (ar *attributeResource) Evaluate(resource pdata.Resource) bool {
	return ar.expr.Evaluate(resource.Attributes())
}
