package queuebatch

import "sync"

var nodePool = sync.Pool{
	New: func() any { return &lruNode{} },
}

type lruNode struct {
	data string
	l    *lruNode
	r    *lruNode
}

type lru struct {
	km map[string]*lruNode
	h  *lruNode
	t  *lruNode
}

func newLRU() *lru {
	l := &lru{
		km: make(map[string]*lruNode),
		h:  &lruNode{},
		t:  &lruNode{},
	}
	l.h.r = l.t
	l.t.l = l.h
	return l
}

func (lru *lru) access(key string) {
	node, ok := lru.km[key]
	if !ok {
		node = nodePool.Get().(*lruNode)
		node.data = key
		node.l = nil
		node.r = nil
		lru.km[key] = node
	} else {
		// Skip if already MRU (right after head sentinel)
		if node.l == lru.h {
			return
		}
		// Remove from current position
		node.l.r = node.r
		node.r.l = node.l
	}
	// Insert at head (MRU position)
	node.r = lru.h.r
	node.l = lru.h
	lru.h.r.l = node
	lru.h.r = node
}

// Method for testing.
func (lru *lru) keyToEvict() string {
	if lru.t.l == lru.h {
		return "" // empty
	}
	return lru.t.l.data
}

func (lru *lru) evictLRU() string {
	if lru.t.l == lru.h {
		return ""
	}
	node := lru.t.l
	node.l.r = lru.t
	lru.t.l = node.l
	delete(lru.km, node.data)
	key := node.data
	// Clear references to help GC
	node.data = ""
	node.l = nil
	node.r = nil
	nodePool.Put(node)
	return key
}

func (lru *lru) evict(key string) {
	node, ok := lru.km[key]
	if !ok {
		return
	}
	node.l.r = node.r
	node.r.l = node.l
	delete(lru.km, key)
	// Clear references to help GC
	node.data = ""
	node.l = nil
	node.r = nil
	nodePool.Put(node)
}

func (lru *lru) len() int {
	return len(lru.km)
}
