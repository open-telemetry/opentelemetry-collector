package rules

func rulesToMatcher(rules Rules) *RulesMatcher {
	expressions, index, defaultRule := mapExpressionsToRules(rules)
	matchers := expressionsToMatchers(expressions)

	engine := NewEarlyEngine(rules, index)
	rulesMatcher := &RulesMatcher{
		Matchers: matchers,
		Engine:   engine,
		Default:  defaultRule,
	}
	return rulesMatcher
}

func mapExpressionsToRules(rules Rules) (expressions []Expression, index [][]RuleID, defaultRule *RuleID) {
	var defaultRuleID RuleID = 0
	var defaultRuleExists bool
	expressionRules := make(map[Expression][]RuleID)
	for ruleID, rule := range rules {
		if len(rule.Expressions) == 0 {
			defaultRuleID = RuleID(ruleID)
			defaultRuleExists = true
			continue
		}
		for _, expression := range rule.Expressions {
			expressionRules[expression] = append(expressionRules[expression], RuleID(ruleID))
		}
	}
	expressions = make([]Expression, 0, len(expressionRules))
	index = make([][]RuleID, 0, len(expressionRules))
	for expression, expRules := range expressionRules {
		expressions = append(expressions, expression)
		index = append(index, expRules)
	}
	if defaultRuleExists {
		defaultRule = &defaultRuleID
	}
	return expressions, index, defaultRule
}

func expressionsToMatchers(expressions []Expression) map[string][]ExpressionMatcher {
	matchers := make(map[string][]ExpressionMatcher)
	for expressionID, expression := range expressions {
		matchers[expression.Key] = append(matchers[expression.Key], expressionToMatcher(ExpressionID(expressionID), expression))
	}
	return matchers
}

func expressionToMatcher(expressionID ExpressionID, expression Expression) ExpressionMatcher {
	switch expression.Operation {
	case "=":
		return &EqualValueMatcher{Values: map[string]ExpressionID{expression.Value: expressionID}}
	case "exists":
		return &ExistsMatcher{Matcher: expressionID}
	default:
		return nil
	}
}
