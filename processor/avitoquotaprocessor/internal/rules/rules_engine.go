package rules

type groupByAttrs struct {
	system   string
	service  string
	severity string
}

type SSSRulesEngine struct {
	groupedRules map[groupByAttrs]*RulesMatcher
}

type AllowFunc func(rule *Rule, n int) bool

// NewSSSRulesEngine creates a new SSSRulesEngine with the given rules.
// It groups the rules by System, Service, and Severity to optimize the matching process.
// The rules are sorted by priority in descending order: system, service, severity.
// If attributes are matched by some (system, service) and (service, severity),
// but not by (system, service, severity), (system, service) rule will be used.
func NewSSSRulesEngine(rules Rules) *SSSRulesEngine {
	groupedRawRules := groupRules(rules)
	groupedRules := map[groupByAttrs]*RulesMatcher{}
	for groupKey, rules := range groupedRawRules {
		groupedRules[groupKey] = rulesToMatcher(rules)
	}

	return &SSSRulesEngine{
		groupedRules: groupedRules,
	}
}

type KVGetter interface {
	Next() bool
	Get() (key string, value string)
	System() string
	Service() string
	Severity() string
}

func (g *SSSRulesEngine) Match(attrs KVGetter, allow AllowFunc) *Rule {
	systemKey, serviceKey, severityKey := attrs.System(), attrs.Service(), attrs.Severity()

	keys := [...]groupByAttrs{
		{systemKey, serviceKey, severityKey},
		{AnyAttr, serviceKey, severityKey},
		{systemKey, serviceKey, AnyAttr},
		{AnyAttr, serviceKey, AnyAttr},
		{systemKey, AnyAttr, severityKey},
		{systemKey, AnyAttr, AnyAttr},
		{AnyAttr, AnyAttr, severityKey},
		{AnyAttr, AnyAttr, AnyAttr},
	}

	for _, groupKey := range keys {
		matcher, ok := g.groupedRules[groupKey]
		if !ok {
			continue
		}
		matcher.Reset()
		for attrs.Next() {
			k, v := attrs.Get()
			matcher.Match(k, v)
		}
		matchedIDs := matcher.GetMatched()
		for _, ruleID := range matchedIDs {
			r := matcher.GetRule(ruleID)
			if r != nil && allow(r, 1) {
				return r
			}
		}
		r := matcher.GetDefaultRule()
		if r != nil && allow(r, 1) {
			return r
		}
	}
	return nil
}

func groupRules(rules Rules) map[groupByAttrs]Rules {
	groupedRules := map[groupByAttrs]Rules{}
	for _, rule := range rules {
		expressions := []Expression{}
		systemKey, serviceKey, severityKey := AnyAttr, AnyAttr, AnyAttr
		for _, expression := range rule.Expressions {
			switch e := expression; {
			case e.Key == "system" && e.Operation == "=":
				systemKey = e.Value
			case e.Key == "service" && e.Operation == "=":
				serviceKey = e.Value
			case e.Key == "severity" && e.Operation == "=":
				severityKey = e.Value
			default:
				expressions = append(expressions, e)
			}
		}
		rule.Expressions = expressions
		groupKey := groupByAttrs{systemKey, serviceKey, severityKey}
		groupedRules[groupKey] = append(groupedRules[groupKey], rule)
	}
	return groupedRules
}
