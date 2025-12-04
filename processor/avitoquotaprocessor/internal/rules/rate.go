package rules

func RulesToRate(rules Rules) map[string]int64 {
	rate := make(map[string]int64, len(rules))
	for _, r := range rules {
		rate[r.ID] = int64(r.RatePerSecond)
	}
	return rate
}

func RulesToStorageRate(rules Rules) map[string]int64 {
	rate := make(map[string]int64, len(rules))
	for _, r := range rules {
		rate[r.ID] = int64(r.StorageRatePerDay)
	}
	return rate
}
