package queuebatch

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLru(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	require.Equal(t, "1", lruKeys.keyToEvict())
	lruKeys.access("2")
	require.Equal(t, "1", lruKeys.keyToEvict())
	lruKeys.access("1")
	require.Equal(t, "2", lruKeys.keyToEvict())
}

func TestLruEmpty(t *testing.T) {
	lruKeys := newLRU()
	require.Equal(t, "", lruKeys.keyToEvict())
}

func TestLruEvictionOrder(t *testing.T) {
	lruKeys := newLRU()
	// Add keys in order 1, 2, 3, 4
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")
	lruKeys.access("4")

	// Oldest (least recently used) should be "1"
	require.Equal(t, "1", lruKeys.keyToEvict())

	// Access "1" again - now "2" should be LRU
	lruKeys.access("1")
	require.Equal(t, "2", lruKeys.keyToEvict())

	// Access "2" - now "3" should be LRU
	lruKeys.access("2")
	require.Equal(t, "3", lruKeys.keyToEvict())
}

func TestLruAccessMiddleElement(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")

	// Access middle element "2" - should not change LRU (still "1")
	lruKeys.access("2")
	require.Equal(t, "1", lruKeys.keyToEvict())

	// Access "1" - now "3" becomes LRU
	lruKeys.access("1")
	require.Equal(t, "3", lruKeys.keyToEvict())
}

func TestLruEvictTail(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")

	// Evict tail (LRU key "1")
	require.Equal(t, "1", lruKeys.keyToEvict())
	lruKeys.evict("1")
	require.Equal(t, "2", lruKeys.keyToEvict())

	// Evict new tail
	lruKeys.evict("2")
	require.Equal(t, "3", lruKeys.keyToEvict())
}

func TestLruEvictLRU(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")

	// evictLRU should return and remove the LRU key
	require.Equal(t, "1", lruKeys.evictLRU())
	require.Equal(t, "2", lruKeys.keyToEvict())

	require.Equal(t, "2", lruKeys.evictLRU())
	require.Equal(t, "3", lruKeys.keyToEvict())

	require.Equal(t, "3", lruKeys.evictLRU())
	require.Equal(t, "", lruKeys.keyToEvict())

	// evictLRU on empty should return ""
	require.Equal(t, "", lruKeys.evictLRU())
}

func TestLruEvictHead(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")

	// Evict head (MRU key "3")
	lruKeys.evict("3")
	require.Equal(t, "1", lruKeys.keyToEvict()) // tail unchanged

	// Access and verify structure still works
	lruKeys.access("1")
	require.Equal(t, "2", lruKeys.keyToEvict())
}

func TestLruEvictMiddle(t *testing.T) {
	lruKeys := newLRU()
	lruKeys.access("1")
	lruKeys.access("2")
	lruKeys.access("3")
	lruKeys.access("4")

	// Evict middle element "2"
	lruKeys.evict("2")
	require.Equal(t, "1", lruKeys.keyToEvict()) // tail unchanged

	// Evict another middle element "3"
	lruKeys.evict("3")
	require.Equal(t, "1", lruKeys.keyToEvict()) // tail still "1"

	// Verify structure by accessing "1" - now "4" is LRU
	lruKeys.access("1")
	require.Equal(t, "4", lruKeys.keyToEvict())
}

func TestLruRandomAccessAndEvict(t *testing.T) {
	rng := rand.New(rand.NewSource(123)) // fixed seed for reproducibility
	lruKeys := newLRU()

	// Reference list to track LRU order (index 0 = LRU, last index = MRU)
	var order []string

	refAccess := func(key string) {
		for i, k := range order {
			if k == key {
				order = append(order[:i], order[i+1:]...)
				break
			}
		}
		order = append(order, key)
	}

	refEvict := func(key string) {
		for i, k := range order {
			if k == key {
				order = append(order[:i], order[i+1:]...)
				return
			}
		}
	}

	numKeys := 10
	numOperations := 10000

	for i := 0; i < numOperations; i++ {
		key := strconv.Itoa(rng.Intn(numKeys))

		// 70% access, 30% evict
		if rng.Float32() < 0.7 {
			lruKeys.access(key)
			refAccess(key)
		} else if len(order) > 0 {
			lruKeys.evict(key)
			refEvict(key)
		}

		// Verify LRU key matches reference
		if len(order) == 0 {
			require.Equal(t, "", lruKeys.keyToEvict(), "expected empty at operation %d", i)
		} else {
			expectedLRU := order[0]
			require.Equal(t, expectedLRU, lruKeys.keyToEvict(), "mismatch at operation %d", i)
		}
	}
}

func TestLruRandomAccessAndEvictLRU(t *testing.T) {
	rng := rand.New(rand.NewSource(456)) // fixed seed for reproducibility
	lruKeys := newLRU()

	// Reference list to track LRU order (index 0 = LRU, last index = MRU)
	var order []string

	refAccess := func(key string) {
		for i, k := range order {
			if k == key {
				order = append(order[:i], order[i+1:]...)
				break
			}
		}
		order = append(order, key)
	}

	refEvictLRU := func() string {
		if len(order) == 0 {
			return ""
		}
		evicted := order[0]
		order = order[1:]
		return evicted
	}

	numKeys := 10
	numOperations := 10000

	for i := 0; i < numOperations; i++ {
		key := strconv.Itoa(rng.Intn(numKeys))

		// 70% access, 30% evictLRU
		if rng.Float32() < 0.7 {
			lruKeys.access(key)
			refAccess(key)
		} else if len(order) > 0 {
			actualEvicted := lruKeys.evictLRU()
			expectedEvicted := refEvictLRU()
			require.Equal(t, expectedEvicted, actualEvicted, "evictLRU mismatch at operation %d", i)
		}

		// Verify LRU key matches reference
		if len(order) == 0 {
			require.Equal(t, "", lruKeys.keyToEvict(), "expected empty at operation %d", i)
		} else {
			expectedLRU := order[0]
			require.Equal(t, expectedLRU, lruKeys.keyToEvict(), "keyToEvict mismatch at operation %d", i)
		}
	}
}

func TestLruRandomAccess(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) // fixed seed for reproducibility
	lruKeys := newLRU()

	// Reference list to track LRU order (index 0 = LRU, last index = MRU)
	var order []string

	// Helper to simulate access in reference list
	refAccess := func(key string) {
		// Remove key if it exists
		for i, k := range order {
			if k == key {
				order = append(order[:i], order[i+1:]...)
				break
			}
		}
		// Add to end (most recently used)
		order = append(order, key)
	}

	// Perform random accesses
	numKeys := 10
	numOperations := 10000

	for i := 0; i < numOperations; i++ {
		key := strconv.Itoa(rng.Intn(numKeys))

		// Access in both implementations
		lruKeys.access(key)
		refAccess(key)

		// Verify LRU key matches reference
		expectedLRU := order[0]
		if expectedLRU != lruKeys.keyToEvict() {
			require.Equal(t, expectedLRU, lruKeys.keyToEvict(), "mismatch at operation %d after accessing key %s", i, key)
		}
		require.Equal(t, expectedLRU, lruKeys.keyToEvict(), "mismatch at operation %d after accessing key %s", i, key)
	}
}
