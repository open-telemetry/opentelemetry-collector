package rules

import (
	"reflect"
	"sort"
	"testing"
)

// Helper to sort RuleID slices for comparison
func sortRuleIDs(slice []RuleID) {
	sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
}

// Helper to sort slices of RuleID slices for comparison
// Note: This sorts the inner slices and then the outer slice based on the first element of inner slices (if present)
// This is a simplistic sort for [][]RuleID for test comparison purposes.
func sortRuleIDMatrix(matrix [][]RuleID) {
	for _, row := range matrix {
		sortRuleIDs(row)
	}
	sort.Slice(matrix, func(i, j int) bool {
		if len(matrix[i]) == 0 && len(matrix[j]) == 0 {
			return false
		}
		if len(matrix[i]) == 0 {
			return true // empty slices first
		}
		if len(matrix[j]) == 0 {
			return false
		}
		return matrix[i][0] < matrix[j][0] // Simplistic comparison
	})
}

// Helper to sort Expression slices for comparison (e.g., by Key then Value)
func sortExpressions(expressions []Expression) {
	sort.Slice(expressions, func(i, j int) bool {
		if expressions[i].Key != expressions[j].Key {
			return expressions[i].Key < expressions[j].Key
		}
		if expressions[i].Operation != expressions[j].Operation {
			return expressions[i].Operation < expressions[j].Operation
		}
		return expressions[i].Value < expressions[j].Value
	})
}

func TestMapExpressionsToRules(t *testing.T) {
	rule0 := &Rule{ID: "R0", Expressions: []Expression{{Key: "k1", Operation: "=", Value: "v1"}}}                                           // RuleID 0
	rule1 := &Rule{ID: "R1", Expressions: []Expression{{Key: "k2", Operation: "=", Value: "v2"}, {Key: "k1", Operation: "=", Value: "v1"}}} // RuleID 1
	rule2 := &Rule{ID: "R2", Expressions: []Expression{}}                                                                                   // RuleID 2 (default rule)
	rule3 := &Rule{ID: "R3", Expressions: []Expression{{Key: "k3", Operation: "=", Value: "v3"}}}                                           // RuleID 3

	tests := []struct {
		name              string
		rules             Rules
		wantExpressions   []Expression
		wantIndex         [][]RuleID
		wantDefaultRuleID *RuleID
	}{
		{
			name:              "no rules",
			rules:             Rules{},
			wantExpressions:   []Expression{},
			wantIndex:         [][]RuleID{},
			wantDefaultRuleID: nil,
		},
		{
			name:              "single rule with one expression",
			rules:             Rules{rule0}, // R0: {k1=v1}
			wantExpressions:   []Expression{{Key: "k1", Operation: "=", Value: "v1"}},
			wantIndex:         [][]RuleID{{0}},
			wantDefaultRuleID: nil,
		},
		{
			name:  "multiple rules, shared expressions, one default",
			rules: Rules{rule0, rule1, rule2, rule3}, // R0:{k1=v1}, R1:{k2=v2, k1=v1}, R2:{}, R3:{k3=v3}
			// Expected unique expressions: {k1=v1}, {k2=v2}, {k3=v3} (order might vary)
			// Expected index mapping (rule IDs):
			// {k1=v1} -> [0, 1]
			// {k2=v2} -> [1]
			// {k3=v3} -> [3]
			wantExpressions: []Expression{
				{Key: "k1", Operation: "=", Value: "v1"},
				{Key: "k2", Operation: "=", Value: "v2"},
				{Key: "k3", Operation: "=", Value: "v3"},
			},
			wantIndex: [][]RuleID{
				{0, 1}, // For k1=v1
				{1},    // For k2=v2
				{3},    // For k3=v3
			},
			wantDefaultRuleID: func() *RuleID { rID := RuleID(2); return &rID }(),
		},
		{
			name:              "only a default rule",
			rules:             Rules{rule2}, // R2: {}
			wantExpressions:   []Expression{},
			wantIndex:         [][]RuleID{},
			wantDefaultRuleID: func() *RuleID { rID := RuleID(0); return &rID }(), // Rule2 is at index 0 here
		},
		{
			name: "multiple default rules (last one wins)",
			rules: Rules{
				{ID: "D1", Expressions: []Expression{}}, // RuleID 0
				{ID: "D2", Expressions: []Expression{}}, // RuleID 1
			},
			wantExpressions:   []Expression{},
			wantIndex:         [][]RuleID{},
			wantDefaultRuleID: func() *RuleID { rID := RuleID(1); return &rID }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExpressions, gotIndex, gotDefaultRuleID := mapExpressionsToRules(tt.rules)

			// Sort results for stable comparison as map iteration order is not guaranteed
			sortExpressions(gotExpressions)
			sortExpressions(tt.wantExpressions) // Ensure want is sorted too for DeepEqual

			// Create a map from expression to its original index in tt.wantExpressions
			// to correctly align gotIndex with tt.wantIndex before sorting tt.wantIndex.
			exprToSortableIndex := make(map[Expression]int)
			for i, expr := range tt.wantExpressions {
				exprToSortableIndex[expr] = i
			}

			// Sort gotIndex based on the sorted order of gotExpressions
			// This requires pairing gotExpressions with gotIndex before sorting
			type exprIndexPair struct {
				expr  Expression
				rules []RuleID
			}
			pairs := make([]exprIndexPair, len(gotExpressions))
			for i := range gotExpressions {
				pairs[i] = exprIndexPair{expr: gotExpressions[i], rules: gotIndex[i]}
			}
			sort.Slice(pairs, func(i, j int) bool {
				// Use the same sorting logic as sortExpressions
				if pairs[i].expr.Key != pairs[j].expr.Key {
					return pairs[i].expr.Key < pairs[j].expr.Key
				}
				if pairs[i].expr.Operation != pairs[j].expr.Operation {
					return pairs[i].expr.Operation < pairs[j].expr.Operation
				}
				return pairs[i].expr.Value < pairs[j].expr.Value
			})
			// Reconstruct sorted gotExpressions and gotIndex
			for i, p := range pairs {
				gotExpressions[i] = p.expr
				gotIndex[i] = p.rules
				sortRuleIDs(gotIndex[i]) // Sort inner RuleID slices
			}

			// Sort tt.wantIndex similarly based on tt.wantExpressions
			wantPairs := make([]exprIndexPair, len(tt.wantExpressions))
			for i := range tt.wantExpressions {
				wantPairs[i] = exprIndexPair{expr: tt.wantExpressions[i], rules: tt.wantIndex[i]}
			}
			sort.Slice(wantPairs, func(i, j int) bool {
				if wantPairs[i].expr.Key != wantPairs[j].expr.Key {
					return wantPairs[i].expr.Key < wantPairs[j].expr.Key
				}
				if wantPairs[i].expr.Operation != wantPairs[j].expr.Operation {
					return wantPairs[i].expr.Operation < wantPairs[j].expr.Operation
				}
				return wantPairs[i].expr.Value < wantPairs[j].expr.Value
			})
			for i, p := range wantPairs {
				tt.wantExpressions[i] = p.expr // already sorted
				tt.wantIndex[i] = p.rules
				sortRuleIDs(tt.wantIndex[i])
			}

			if !reflect.DeepEqual(gotExpressions, tt.wantExpressions) {
				t.Errorf("mapExpressionsToRules() gotExpressions = %v, want %v", gotExpressions, tt.wantExpressions)
			}
			sortRuleIDMatrix(gotIndex)
			sortRuleIDMatrix(tt.wantIndex)
			if !reflect.DeepEqual(gotIndex, tt.wantIndex) {
				t.Errorf("mapExpressionsToRules() gotIndex = %v, want %v", gotIndex, tt.wantIndex)
			}
			if !reflect.DeepEqual(gotDefaultRuleID, tt.wantDefaultRuleID) {
				t.Errorf("mapExpressionsToRules() gotDefaultRuleID = %v, want %v", gotDefaultRuleID, tt.wantDefaultRuleID)
			}
		})
	}
}

func TestExpressionToMatcher(t *testing.T) {
	tests := []struct {
		name         string
		expressionID ExpressionID
		expression   Expression
		wantMatcher  ExpressionMatcher // Check type and relevant fields
	}{
		{
			name:         "equal operation",
			expressionID: ExpressionID(1),
			expression:   Expression{Key: "k", Operation: "=", Value: "v"},
			wantMatcher:  &EqualValueMatcher{Values: map[string]ExpressionID{"v": ExpressionID(1)}},
		},
		{
			name:         "exists operation",
			expressionID: ExpressionID(2),
			expression:   Expression{Key: "k", Operation: "exists", Value: ""}, // Value ignored for exists
			wantMatcher:  &ExistsMatcher{Matcher: ExpressionID(2)},
		},
		{
			name:         "unknown operation",
			expressionID: ExpressionID(3),
			expression:   Expression{Key: "k", Operation: "unknown", Value: "v"},
			wantMatcher:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatcher := expressionToMatcher(tt.expressionID, tt.expression)
			if !reflect.DeepEqual(gotMatcher, tt.wantMatcher) {
				t.Errorf("expressionToMatcher() gotMatcher = %T %v, want %T %v", gotMatcher, gotMatcher, tt.wantMatcher, tt.wantMatcher)
			}
		})
	}
}

func TestExpressionsToMatchers(t *testing.T) {
	expr1 := Expression{Key: "k1", Operation: "=", Value: "v1"}    // Mapped to ExpressionID 0
	expr2 := Expression{Key: "k1", Operation: "exists", Value: ""} // Mapped to ExpressionID 1 (same key as expr1)
	expr3 := Expression{Key: "k2", Operation: "=", Value: "v2"}    // Mapped to ExpressionID 2

	expressions := []Expression{expr1, expr2, expr3}

	// Expected structure:
	// "k1" -> [EqualValueMatcher for v1 (ExprID 0), ExistsMatcher (ExprID 1)]
	// "k2" -> [EqualValueMatcher for v2 (ExprID 2)]

	gotMatchersMap := expressionsToMatchers(expressions)

	// Check k1
	if matchersK1, ok := gotMatchersMap["k1"]; !ok || len(matchersK1) != 2 {
		t.Errorf("expressionsToMatchers() for key 'k1': got %v, want 2 matchers", matchersK1)
	} else {
		// Check types and basic properties. Order might vary.
		foundEqualK1, foundExistsK1 := false, false
		for _, m := range matchersK1 {
			if eqMatcher, isEq := m.(*EqualValueMatcher); isEq {
				if id, ok := eqMatcher.Values["v1"]; ok && id == ExpressionID(0) {
					foundEqualK1 = true
				}
			}
			if exMatcher, isEx := m.(*ExistsMatcher); isEx {
				if exMatcher.Matcher == ExpressionID(1) {
					foundExistsK1 = true
				}
			}
		}
		if !foundEqualK1 {
			t.Errorf("expressionsToMatchers() for key 'k1': missing EqualValueMatcher for v1 (ExprID 0)")
		}
		if !foundExistsK1 {
			t.Errorf("expressionsToMatchers() for key 'k1': missing ExistsMatcher (ExprID 1)")
		}
	}

	// Check k2
	if matchersK2, ok := gotMatchersMap["k2"]; !ok || len(matchersK2) != 1 {
		t.Errorf("expressionsToMatchers() for key 'k2': got %v, want 1 matcher", matchersK2)
	} else {
		if eqMatcher, isEq := matchersK2[0].(*EqualValueMatcher); !isEq {
			t.Errorf("expressionsToMatchers() for key 'k2': want EqualValueMatcher, got %T", matchersK2[0])
		} else {
			if id, ok := eqMatcher.Values["v2"]; !ok || id != ExpressionID(2) {
				t.Errorf("expressionsToMatchers() for key 'k2': EqualValueMatcher has %v, want map[v2:%d]", eqMatcher.Values, ExpressionID(2))
			}
		}
	}

	// Check no other keys
	if len(gotMatchersMap) != 2 {
		t.Errorf("expressionsToMatchers() map length = %d, want 2. Map: %v", len(gotMatchersMap), gotMatchersMap)
	}
}

func TestRulesToMatcher(t *testing.T) {
	ruleWithExpr := &Rule{ID: "R1", Expressions: []Expression{{Key: "k", Operation: "=", Value: "v"}}}
	defaultRule := &Rule{ID: "R_Default", Expressions: []Expression{}}

	tests := []struct {
		name            string
		rules           Rules
		wantMatcherKeys []string    // Keys expected in RulesMatcher.Matchers
		wantEngineType  interface{} // e.g. *EarlyEngine
		wantDefaultID   *RuleID
	}{
		{
			name:            "single rule with expression",
			rules:           Rules{ruleWithExpr},
			wantMatcherKeys: []string{"k"},
			wantEngineType:  (*EarlyEngine)(nil),
			wantDefaultID:   nil,
		},
		{
			name:            "only default rule",
			rules:           Rules{defaultRule},
			wantMatcherKeys: []string{}, // No explicit matchers
			wantEngineType:  (*EarlyEngine)(nil),
			wantDefaultID:   func() *RuleID { id := RuleID(0); return &id }(),
		},
		{
			name:            "rule with expression and a default rule",
			rules:           Rules{ruleWithExpr, defaultRule}, // R1 (idx 0), R_Default (idx 1)
			wantMatcherKeys: []string{"k"},
			wantEngineType:  (*EarlyEngine)(nil),
			wantDefaultID:   func() *RuleID { id := RuleID(1); return &id }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rulesMatcher := rulesToMatcher(tt.rules)

			if rulesMatcher == nil {
				t.Fatalf("rulesToMatcher() returned nil")
			}

			// Check Matchers keys
			gotKeys := make([]string, 0, len(rulesMatcher.Matchers))
			for k := range rulesMatcher.Matchers {
				gotKeys = append(gotKeys, k)
			}
			sort.Strings(gotKeys)
			sort.Strings(tt.wantMatcherKeys)
			if !reflect.DeepEqual(gotKeys, tt.wantMatcherKeys) {
				t.Errorf("rulesToMatcher().Matchers keys = %v, want %v", gotKeys, tt.wantMatcherKeys)
			}

			// Check Engine type
			if reflect.TypeOf(rulesMatcher.Engine) != reflect.TypeOf(tt.wantEngineType) {
				t.Errorf("rulesToMatcher().Engine type = %T, want %T", rulesMatcher.Engine, tt.wantEngineType)
			}

			// Check Default rule ID
			if !reflect.DeepEqual(rulesMatcher.Default, tt.wantDefaultID) {
				if rulesMatcher.Default != nil && tt.wantDefaultID != nil {
					t.Errorf("rulesToMatcher().Default ID = *%v, want *%v", *rulesMatcher.Default, *tt.wantDefaultID)
				} else {
					t.Errorf("rulesToMatcher().Default ID = %v, want %v", rulesMatcher.Default, tt.wantDefaultID)
				}
			}

			// Further checks on engine content if needed, e.g., rules count
			if earlyEngine, ok := rulesMatcher.Engine.(*EarlyEngine); ok {
				if len(earlyEngine.rules) != len(tt.rules) {
					t.Errorf("Engine rules count = %d, want %d", len(earlyEngine.rules), len(tt.rules))
				}
			}
		})
	}
}
