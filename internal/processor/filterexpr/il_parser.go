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
	"errors"

	"github.com/alecthomas/participle/v2"
)

type ilsExpression struct {
	LogicalOr *ilsLogicalOr `parser:"@@"`
}

func (expr *ilsExpression) toIlExpr() (IlExpr, error) {
	if expr.LogicalOr == nil {
		return nil, nil
	}
	return expr.LogicalOr.toIlExpr()
}

type ilsLogicalOr struct {
	LogicalAnd *ilsLogicalAnd `parser:" @@"`
	Op         string         `parser:" [ @( '|' '|' ) "`
	Next       *ilsLogicalOr  `parser:"   @@ ] "`
}

func (lor *ilsLogicalOr) toIlExpr() (IlExpr, error) {
	if lor.Op != "" {
		expr1, err := lor.LogicalAnd.toIlExpr()
		if err != nil {
			return nil, err
		}
		var expr2 IlExpr
		expr2, err = lor.Next.toIlExpr()
		if err != nil {
			return nil, err
		}
		return newOrIlExpr(expr1, expr2), nil
	}
	return lor.LogicalAnd.toIlExpr()
}

type ilsLogicalAnd struct {
	Equality *ilsEquality   `parser:" @@ "`
	Op       string         `parser:" [ @( '&' '&') "`
	Next     *ilsLogicalAnd `parser:"  @@ ] "`
}

func (land *ilsLogicalAnd) toIlExpr() (IlExpr, error) {
	if land.Op != "" {
		expr1, err := land.Equality.toIlExpr()
		if err != nil {
			return nil, err
		}
		var expr2 IlExpr
		expr2, err = land.Next.toIlExpr()
		if err != nil {
			return nil, err
		}
		return newAndIlExpr(expr1, expr2), nil
	}
	return land.Equality.toIlExpr()
}

type ilsEquality struct {
	Field   string    `parser:" ( @Ident"`
	Op      string    `parser:"   @( '!' '=' | '=' '=' | '~' '=') "`
	Literal string    `parser:"  @String ) "`
	Unary   *ilsUnary `parser:"| @@ "`
}

func (eq *ilsEquality) toIlExpr() (IlExpr, error) {
	switch eq.Op {
	case "!=":
		switch eq.Field {
		case "Name":
			return newNameIlExpr(newNotEqualStringExpr(eq.Literal)), nil
		case "Version":
			return newVersionIlExpr(newNotEqualStringExpr(eq.Literal)), nil
		}
		return nil, errors.New("unrecognized field for equality operation")
	case "==":
		switch eq.Field {
		case "Name":
			return newNameIlExpr(newEqualStringExpr(eq.Literal)), nil
		case "Version":
			return newVersionIlExpr(newEqualStringExpr(eq.Literal)), nil
		}
		return nil, errors.New("unrecognized field for equality operation")
	case "~=":
		switch eq.Field {
		case "Name":
			expr, err := newRegexpStringExpr(eq.Literal)
			if err != nil {
				return nil, err
			}
			return newNameIlExpr(expr), nil
		case "Version":
			expr, err := newRegexpStringExpr(eq.Literal)
			if err != nil {
				return nil, err
			}
			return newVersionIlExpr(expr), nil
		}
		return nil, errors.New("unrecognized field for equality operation")
	}
	return eq.Unary.toIlExpr()
}

type ilsUnary struct {
	Op            string         `parser:" ( @( '!' ) "`
	Unary         *ilsUnary      `parser:"   @@ ) "`
	SubExpression *ilsExpression `parser:" | '(' @@ ')' "`
}

func (una *ilsUnary) toIlExpr() (IlExpr, error) {
	if una.Op != "" {
		expr, err := una.Unary.toIlExpr()
		if err != nil {
			return nil, err
		}
		return newNotIlExpr(expr), nil
	}
	return una.SubExpression.toIlExpr()
}

var ilsParser = participle.MustBuild(&ilsExpression{}, participle.UseLookahead(10), participle.Unquote())

// CompileIlExpr compiles an expression and returns a IlExpr that can be used
// to evaluate the expression against a pdata.InstrumentationLibrary.
//
// The expression uses the following grammar:
// expression     → logicalOr ;
// logicalOr      → logicalAnd ( ( "||" ) logicalAnd )* ;
// logicalAnd     → equality ( ( "&&" ) equality )* ;
// equality       → ( (Name|Version) ( "!=" | "==" | "~=" ) STRING )* | unary ;
// unary          → ( ( "!" ) unary )* | "(" expression ")" ;
//
// Available Fields:
//   * Name - access the name field of the instrumentation library.
//   * Version - access the version field of the instrumentation library.
//
// All equalities must start use the field as the first term.
func CompileIlExpr(exprStr string) (IlExpr, error) {
	if exprStr == "" {
		return nil, nil
	}
	expr := &ilsExpression{}
	if err := ilsParser.ParseString("", exprStr, expr); err != nil {
		return nil, err
	}
	return expr.toIlExpr()
}
