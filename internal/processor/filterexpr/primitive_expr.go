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

type StringExpr interface {
	Evaluate(val string) bool
}

type strictStringExpr string

func NewStrictStringExpr(val string) StringExpr {
	return strictStringExpr(val)
}

func (ssp strictStringExpr) Evaluate(val string) bool {
	return string(ssp) == val
}

type regexpStringExpr struct {
	re *regexp.Regexp
}

func NewRegexpStringExpr(val string) (StringExpr, error) {
	re, err := regexp.Compile(val)
	if err != nil {
		return nil, err
	}
	return &regexpStringExpr{re: re}, nil
}

func (rsp *regexpStringExpr) Evaluate(val string) bool {
	return rsp.re.MatchString(val)
}
