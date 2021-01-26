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

type resourceExpression struct {
	LogicalOr *resourceLogicalOr `parser:"@@"`
}

func (expr *resourceExpression) toResourceExpr() (ResourceExpr, error) {
	if expr.LogicalOr == nil {
		return nil, nil
	}
	return expr.LogicalOr.toResourceExpr()
}

type resourceLogicalOr struct {
	LogicalAnd *resourceLogicalAnd `parser:"@@"`
	Op         string              `parser:" [ @( '|' '|' ) "`
	Next       *resourceLogicalOr  `parser:"   @@ ] "`
}

func (lor *resourceLogicalOr) toResourceExpr() (ResourceExpr, error) {
	if lor.Op != "" {
		expr1, err := lor.LogicalAnd.toResourceExpr()
		if err != nil {
			return nil, err
		}
		var expr2 ResourceExpr
		expr2, err = lor.Next.toResourceExpr()
		if err != nil {
			return nil, err
		}
		return newOrResourceExpr(expr1, expr2), nil
	}
	return lor.LogicalAnd.toResourceExpr()
}

type resourceLogicalAnd struct {
	Equality *resourceEquality   `parser:"@@"`
	Op       string              `parser:" [ @( '&' '&') "`
	Next     *resourceLogicalAnd `parser:"   @@ ] "`
}

func (land *resourceLogicalAnd) toResourceExpr() (ResourceExpr, error) {
	if land.Op != "" {
		expr1, err := land.Equality.toResourceExpr()
		if err != nil {
			return nil, err
		}
		var expr2 ResourceExpr
		expr2, err = land.Next.toResourceExpr()
		if err != nil {
			return nil, err
		}
		return newAndResourceExpr(expr1, expr2), nil
	}
	return land.Equality.toResourceExpr()
}

type resourceEquality struct {
	AttributeKey string           `parser:" ( 'Attribute' '(' @String ')'"`
	Op           string           `parser:"   @( '!' '=' | '=' '=' | '~' '=') "`
	Literal      *resourceLiteral `parser:"   @@ ) "`
	Unary        *resourceUnary   `parser:" | @@ "`
}

func (eq *resourceEquality) toResourceExpr() (ResourceExpr, error) {
	switch eq.Op {
	case "!=":
		switch {
		case eq.Literal.String != nil:
			return newAttributesResourceExpr(newStringAttributeExpr(eq.AttributeKey, newNotEqualStringExpr(*eq.Literal.String))), nil
		case eq.Literal.Int != nil:
			return newAttributesResourceExpr(newIntAttributeExpr(eq.AttributeKey, newNotEqualIntExpr(*eq.Literal.Int))), nil
		case eq.Literal.Double != nil:
			return newAttributesResourceExpr(newDoubleAttributeExpr(eq.AttributeKey, newNotEqualDoubleExpr(*eq.Literal.Double))), nil
		case eq.Literal.Bool != nil:
			return newAttributesResourceExpr(newBoolAttributeExpr(eq.AttributeKey, !(*eq.Literal.Bool))), nil
		}
	case "==":
		switch {
		case eq.Literal.String != nil:
			return newAttributesResourceExpr(newStringAttributeExpr(eq.AttributeKey, newEqualStringExpr(*eq.Literal.String))), nil
		case eq.Literal.Int != nil:
			return newAttributesResourceExpr(newIntAttributeExpr(eq.AttributeKey, newEqualIntExpr(*eq.Literal.Int))), nil
		case eq.Literal.Double != nil:
			return newAttributesResourceExpr(newDoubleAttributeExpr(eq.AttributeKey, newEqualDoubleExpr(*eq.Literal.Double))), nil
		case eq.Literal.Bool != nil:
			return newAttributesResourceExpr(newBoolAttributeExpr(eq.AttributeKey, *eq.Literal.Bool)), nil
		}
	case "~=":
		if eq.Literal.String == nil {
			return nil, errors.New("regexp supports only string")
		}
		expr, err := newRegexpStringExpr(*eq.Literal.String)
		if err != nil {
			return nil, err
		}
		return newAttributesResourceExpr(newStringAttributeExpr(eq.AttributeKey, expr)), nil
	}
	return eq.Unary.toResourceExpr()
}

type resourceUnary struct {
	Op      string           `parser:" ( @( '!' ) "`
	Unary   *resourceUnary   `parser:"   @@ ) "`
	Primary *resourcePrimary `parser:" | @@ "`
}

func (una *resourceUnary) toResourceExpr() (ResourceExpr, error) {
	if una.Op != "" {
		expr, err := una.Unary.toResourceExpr()
		if err != nil {
			return nil, err
		}
		return newNotResourceExpr(expr), nil
	}
	return una.Primary.toResourceExpr()
}

type resourcePrimary struct {
	HasAttributeKey *string             `parser:" ( 'HasAttribute' '(' @String ')' ) "`
	SubExpression   *resourceExpression `parser:" | '(' @@ ')' "`
}

func (una *resourcePrimary) toResourceExpr() (ResourceExpr, error) {
	if una.HasAttributeKey != nil {
		return newAttributesResourceExpr(newHasAttributeExpr(*una.HasAttributeKey)), nil
	}
	return una.SubExpression.toResourceExpr()
}

type resourceLiteral struct {
	String *string  `parser:" @String "`
	Double *float64 `parser:" | @Float "`
	Int    *int64   `parser:" | @Int "`
	Bool   *bool    `parser:" | @('true' | 'false') "`
}

var resourceParser = participle.MustBuild(&resourceExpression{}, participle.UseLookahead(10), participle.Unquote())

// CompileResourceExpr compiles an expression and returns a ResourceExpr that can be used
// to evaluate the expression against a pdata.Resource.
//
// The expression uses the following grammar:
// expression     → logicalOr ;
// logicalOr      → logicalAnd ( ( "||" ) logicalAnd )* ;
// logicalAnd     → equality ( ( "&&" ) equality )* ;
// equality       → ( Attribute(STRING) ( "!=" | "==" | "~=" ) (STRING|FLOAT|INT|BOOL) )* | unary ;
// unary          → ( ( "!" ) unary )* | primary ;
// primary        → ( HasAttribute(STRING) )* | "(" expression ")" ;
//
// Available Fields:
//   * HasAttribute("key") - function that returns if an attribute exists with the specified key.
//   * Attribute("key") - function that returns the value of a resource attribute with the specified key.
//     Attributes can have different value types, the type of the value to compare with is determined
//     by the type of the second term of the equality.
//
// All equalities must start use the field as the first term.
//
// Caveat: For floats always use "." even if the value is integer (e.g. "2.0").
func CompileResourceExpr(exprStr string) (ResourceExpr, error) {
	if exprStr == "" {
		return nil, nil
	}
	expr := &resourceExpression{}
	if err := resourceParser.ParseString("", exprStr, expr); err != nil {
		return nil, err
	}
	return expr.toResourceExpr()
}
