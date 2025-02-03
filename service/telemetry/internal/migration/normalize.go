package migration

import "strings"

func normalizeEndpoint(endpoint string) *string {
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		endpoint = "http://" + endpoint
	}
	return &endpoint
}
