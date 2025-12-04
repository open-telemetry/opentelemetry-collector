package rules

const (
	SystemResourceAttribute  = "system"
	ServiceResourceAttribute = "service.name"

	ResourceType = "resource"
	ScopeType    = "scope"
	LogType      = "log"
)

type Attributes struct {
	resourceAttributes [][2]string
	scopeAttributes    [][2]string
	logAttributes      [][2]string

	typ string
	idx int

	system   string
	service  string
	severity string
}

func NewAttributes(resourceAttributes [][2]string, scopeAttributes [][2]string, logAttributes [][2]string, system string, service string, severity string) *Attributes {
	return &Attributes{
		resourceAttributes: resourceAttributes,
		scopeAttributes:    scopeAttributes,
		logAttributes:      logAttributes,
		system:             system,
		service:            service,
		severity:           severity,
		typ:                ResourceType,
		idx:                -1,
	}
}

func (it *Attributes) Next() bool {
	it.idx++
	switch it.typ {
	case ResourceType:
		if it.idx < len(it.resourceAttributes) {
			return true
		} else {
			it.typ = ScopeType
			it.idx = 0
		}
	case ScopeType:
		if it.idx < len(it.scopeAttributes) {
			return true
		} else {
			it.typ = LogType
			it.idx = 0
		}
	case LogType:
		if it.idx < len(it.logAttributes) {
			return true
		} else {
			return false
		}
	default:
		return false
	}
	return false
}

func (it *Attributes) Get() (key string, value string) {
	switch it.typ {
	case ResourceType:
		if it.idx < len(it.resourceAttributes) {
			return it.resourceAttributes[it.idx][0], it.resourceAttributes[it.idx][1]
		}
	case ScopeType:
		if it.idx < len(it.scopeAttributes) {
			return it.scopeAttributes[it.idx][0], it.scopeAttributes[it.idx][1]
		}
	case LogType:
		if it.idx < len(it.logAttributes) {
			return it.logAttributes[it.idx][0], it.logAttributes[it.idx][1]
		}
	default:
		return "", ""
	}
	return "", ""
}

func (it *Attributes) System() string {
	return it.system
}

func (it *Attributes) Service() string {
	return it.service
}

func (it *Attributes) Severity() string {
	return it.severity
}

// --- For tests ---

// MapKVGetter is an iterator wrapper around map[string]string.
type MapKVGetter struct {
	m    map[string]string // the underlying map
	keys []string          // snapshot of keys for deterministic iteration
	idx  int               // current position (–1 before first call to Next)
}

// NewMapKVGetter builds an iterator from a map.
// It copies the keys once so iteration order is stable for the lifetime
// of the iterator (though the order is still arbitrary).
func NewMapKVGetter(m map[string]string) *MapKVGetter {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return &MapKVGetter{m: m, keys: keys, idx: -1}
}

// Next moves to the next element. It returns false when no more elements remain.
func (it *MapKVGetter) Next() bool {
	it.idx++
	return it.idx < len(it.keys)
}

// Get returns the key and value at the current cursor.
// Call it only after a successful Next().
func (it *MapKVGetter) Get() (string, string) {
	if it.idx < 0 || it.idx >= len(it.keys) {
		return "", "" // or panic, depending on taste
	}
	k := it.keys[it.idx]
	return k, it.m[k]
}

func (it *MapKVGetter) System() string {
	res, ok := it.m["system"]
	if !ok {
		return AnyAttr
	}
	return res
}

func (it *MapKVGetter) Service() string {
	res, ok := it.m["service"]
	if !ok {
		return AnyAttr
	}
	return res
}

func (it *MapKVGetter) Severity() string {
	res, ok := it.m["severity"]
	if !ok {
		return AnyAttr
	}
	return res
}
