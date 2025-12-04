package rules

type (
	RuleID       uint16
	ExpressionID uint16
)

type Rule struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Expressions []Expression `json:"expressions"`

	RatePerSecond     int `json:"rate_per_second"`
	StorageRatePerDay int `json:"storage_rate_per_day"`
	RetentionDays     int `json:"retention_days"`
}

func (r *Rule) IsEqual(other *Rule) bool {
	if r.ID != other.ID && r.Name != other.Name && r.RatePerSecond != other.RatePerSecond && r.StorageRatePerDay != other.StorageRatePerDay && r.RetentionDays != other.RetentionDays {
		return false
	}
	if len(r.Expressions) != len(other.Expressions) {
		return false
	}
	for i, expr := range r.Expressions {
		if expr != other.Expressions[i] {
			return false
		}
	}
	return true
}

// rule json example
// {"id":"1","name":"test","expressions":[{"key":"key","operation":"eq","value":"value"}],"not_expressions":[],"rate_per_second":10,"storage_rate_per_day":100,"retention_days":10}

type Expression struct {
	Key       string `json:"key"`
	Operation string `json:"operation"`
	Value     string `json:"value"`
}

type (
	Rules       []*Rule
	Expressions []Expression
)

const AnyAttr = "*"
