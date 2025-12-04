package rules

import (
	"reflect"
	"testing"
)

// --- MapKVGetter Tests ---

func TestNewMapKVGetter(t *testing.T) {
	m := map[string]string{"k1": "v1", "k2": "v2"}
	getter := NewMapKVGetter(m)

	if getter.idx != -1 {
		t.Errorf("NewMapKVGetter idx = %d, want -1", getter.idx)
	}
	if len(getter.keys) != len(m) {
		t.Errorf("NewMapKVGetter len(keys) = %d, want %d", len(getter.keys), len(m))
	}
	// Check if all keys from m are in getter.keys
	for k := range m {
		found := false
		for _, gk := range getter.keys {
			if k == gk {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("NewMapKVGetter key %s missing from getter.keys", k)
		}
	}
}

func TestMapKVGetter_Next_Get(t *testing.T) {
	tests := []struct {
		name      string
		m         map[string]string
		wantCount int // Number of expected Next()=true calls
		// We can't easily predict order for map, so we check count and content
	}{
		{
			name:      "empty map",
			m:         map[string]string{},
			wantCount: 0,
		},
		{
			name:      "single item map",
			m:         map[string]string{"k1": "v1"},
			wantCount: 1,
		},
		{
			name:      "multiple item map",
			m:         map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := NewMapKVGetter(tt.m)
			// Test Get before Next
			k, v := getter.Get()
			if k != "" || v != "" {
				t.Errorf("Get() before Next: got k=%s, v=%s, want \"\", \"\"", k, v)
			}

			retrieved := make(map[string]string)
			count := 0
			for getter.Next() {
				k, v := getter.Get()
				if _, exists := tt.m[k]; !exists {
					t.Errorf("Get() returned unexpected key: %s", k)
				}
				if v != tt.m[k] {
					t.Errorf("Get() for key %s returned value %s, want %s", k, v, tt.m[k])
				}
				retrieved[k] = v
				count++
			}

			if count != tt.wantCount {
				t.Errorf("Number of Next() true calls = %d, want %d", count, tt.wantCount)
			}
			if !reflect.DeepEqual(retrieved, tt.m) && !(len(retrieved) == 0 && len(tt.m) == 0) {
				// Allow empty maps to be considered equal by reflect.DeepEqual if both are empty
				t.Errorf("Retrieved map items = %v, want %v", retrieved, tt.m)
			}

			// Test Get after Next returns false
			kAfterFalse, vAfterFalse := getter.Get()
			if kAfterFalse != "" || vAfterFalse != "" {
				t.Errorf("Get() after Next()=false: got k=%s, v=%s, want \"\", \"\"", kAfterFalse, vAfterFalse)
			}
		})
	}
}

func TestMapKVGetter_System_Service_Severity(t *testing.T) {
	tests := []struct {
		name         string
		m            map[string]string
		wantSystem   string
		wantService  string
		wantSeverity string
	}{
		{
			name:         "all present",
			m:            map[string]string{"system": "sysA", "service": "srvX", "severity": "sev1"},
			wantSystem:   "sysA",
			wantService:  "srvX",
			wantSeverity: "sev1",
		},
		{
			name:         "none present",
			m:            map[string]string{"other": "val"},
			wantSystem:   AnyAttr,
			wantService:  AnyAttr,
			wantSeverity: AnyAttr,
		},
		{
			name:         "some present",
			m:            map[string]string{"system": "sysB", "other": "val"},
			wantSystem:   "sysB",
			wantService:  AnyAttr,
			wantSeverity: AnyAttr,
		},
		{
			name:         "empty map",
			m:            map[string]string{},
			wantSystem:   AnyAttr,
			wantService:  AnyAttr,
			wantSeverity: AnyAttr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := NewMapKVGetter(tt.m)
			if sys := getter.System(); sys != tt.wantSystem {
				t.Errorf("System() = %s, want %s", sys, tt.wantSystem)
			}
			if svc := getter.Service(); svc != tt.wantService {
				t.Errorf("Service() = %s, want %s", svc, tt.wantService)
			}
			if sev := getter.Severity(); sev != tt.wantSeverity {
				t.Errorf("Severity() = %s, want %s", sev, tt.wantSeverity)
			}
		})
	}
}
