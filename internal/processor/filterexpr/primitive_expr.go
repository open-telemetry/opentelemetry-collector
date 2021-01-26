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
	"regexp"
)

type IntExpr interface {
	Evaluate(val int64) bool
}

type notEqualIntExpr int64

func newNotEqualIntExpr(val int64) IntExpr {
	return notEqualIntExpr(val)
}

func (ssp notEqualIntExpr) Evaluate(val int64) bool {
	return int64(ssp) != val
}

type equalIntExpr int64

func newEqualIntExpr(val int64) IntExpr {
	return equalIntExpr(val)
}

func (ssp equalIntExpr) Evaluate(val int64) bool {
	return int64(ssp) == val
}

type DoubleExpr interface {
	Evaluate(val float64) bool
}

type notEqualDoubleExpr float64

func newNotEqualDoubleExpr(val float64) DoubleExpr {
	return notEqualDoubleExpr(val)
}

func (ssp notEqualDoubleExpr) Evaluate(val float64) bool {
	return float64(ssp) != val
}

type equalDoubleExpr float64

func newEqualDoubleExpr(val float64) DoubleExpr {
	return equalDoubleExpr(val)
}

func (ssp equalDoubleExpr) Evaluate(val float64) bool {
	return float64(ssp) == val
}

type StringExpr interface {
	Evaluate(val string) bool
}

type notEqualStringExpr string

func newNotEqualStringExpr(val string) StringExpr {
	return notEqualStringExpr(val)
}

func (ssp notEqualStringExpr) Evaluate(val string) bool {
	return string(ssp) != val
}

type equalStringExpr string

func newEqualStringExpr(val string) StringExpr {
	return equalStringExpr(val)
}

func (ssp equalStringExpr) Evaluate(val string) bool {
	return string(ssp) == val
}

type regexpStringExpr struct {
	re *regexp.Regexp
}

func newRegexpStringExpr(val string) (StringExpr, error) {
	re, err := regexp.Compile(val)
	if err != nil {
		return nil, err
	}
	return &regexpStringExpr{re: re}, nil
}

func (rsp *regexpStringExpr) Evaluate(val string) bool {
	return rsp.re.MatchString(val)
}
