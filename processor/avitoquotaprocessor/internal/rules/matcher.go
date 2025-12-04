package rules

type MatchUpdater interface {
	Update(expressionID ExpressionID) (matched bool)
	GetMatched() []RuleID
	GetRule(ruleID RuleID) *Rule
	Reset()
	ResetMatched()
}

type ExpressionMatcher interface {
	Match(value string) (expressionIDs ExpressionID, expressionMatched bool)
}

type RulesMatcher struct {
	Matchers map[string][]ExpressionMatcher
	Engine   MatchUpdater
	Default  *RuleID
}

func (km *RulesMatcher) Match(key string, value string) (matched bool) {
	matched = false
	matchers, ok := km.Matchers[key]
	if !ok {
		return matched
	}
	for _, matcher := range matchers {
		expressionID, expressionMatched := matcher.Match(value)
		if !expressionMatched {
			continue
		}
		ruleMatched := km.Engine.Update(expressionID)
		matched = matched || ruleMatched
	}
	return matched
}

func (km *RulesMatcher) GetMatched() []RuleID {
	return km.Engine.GetMatched()
}

func (km *RulesMatcher) GetRule(ruleID RuleID) *Rule {
	return km.Engine.GetRule(ruleID)
}

func (km *RulesMatcher) GetDefaultRule() *Rule {
	if km.Default == nil {
		return nil
	}
	return km.Engine.GetRule(*km.Default)
}

func (km *RulesMatcher) ResetMatched() {
	km.Engine.ResetMatched()
}

func (km *RulesMatcher) Reset() {
	km.Engine.Reset()
}

type EqualValueMatcher struct {
	Values map[string]ExpressionID
}

func (m *EqualValueMatcher) Match(value string) (ExpressionID, bool) {
	matched, ok := m.Values[value]
	return matched, ok
}

type ExistsMatcher struct {
	Matcher ExpressionID
}

func (m *ExistsMatcher) Match(value string) (ExpressionID, bool) {
	return m.Matcher, true
}
