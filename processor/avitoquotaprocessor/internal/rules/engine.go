package rules

type EarlyEngine struct {
	expressionRules [][]RuleID // ExpressionID -> []RuleID
	expressionDone  []uint32   // ExpressionID    -> bool

	rules Rules    // RuleID -> Rule
	stamp []uint32 // RuleID -> epoch
	count []uint8  // RuleID -> count

	epoch          uint32
	matchedCount   int
	matchedRuleIDs []RuleID
}

func NewEarlyEngine(rules Rules, index [][]RuleID) *EarlyEngine {
	return &EarlyEngine{
		expressionRules: index,
		expressionDone:  make([]uint32, len(index)),
		rules:           rules,
		stamp:           make([]uint32, len(rules)),
		count:           make([]uint8, len(rules)),

		epoch:          1,
		matchedCount:   0,
		matchedRuleIDs: make([]RuleID, len(rules)),
	}
}

func (e *EarlyEngine) Update(expressionID ExpressionID) (matched bool) {
	matched = false
	if e.expressionDone[expressionID] == e.epoch {
		return matched
	}
	e.expressionDone[expressionID] = e.epoch

	for _, ruleID := range e.expressionRules[expressionID] {
		if e.stamp[ruleID] != e.epoch {
			e.stamp[ruleID], e.count[ruleID] = e.epoch, 1
		} else {
			e.count[ruleID]++
		}
		// TODO: replace with predefined length
		if e.count[ruleID] == uint8(len(e.rules[ruleID].Expressions)) {
			e.matchedRuleIDs[e.matchedCount] = ruleID
			e.matchedCount++
			matched = true
			// we can't return here because we need to update all rules:
			// e.g. rule1: a && b, rule2: a && c, if a is true, we need to update both rules
			// as c and b may be already matched in previous iterations
			// so current iteration may match both rules
			// in some cases, if caller want to get both rules and we do early return here,
			// caller will only get the first rule and miss the second rule forever
		}
	}
	return matched
}

func (e *EarlyEngine) Reset() {
	e.matchedCount = 0
	e.epoch++
}

func (e *EarlyEngine) ResetMatched() {
	e.matchedCount = 0
}

func (e *EarlyEngine) GetMatched() []RuleID {
	if e.matchedCount == 0 {
		return nil
	}
	return e.matchedRuleIDs[:e.matchedCount]
}

func (e *EarlyEngine) GetRule(ruleID RuleID) *Rule {
	//lint:ignore SA4003 Guard if the ruleID is changed to signed int
	if ruleID < 0 || int(ruleID) >= len(e.rules) { //nolint:staticcheck
		return nil
	}
	return e.rules[ruleID]
}
