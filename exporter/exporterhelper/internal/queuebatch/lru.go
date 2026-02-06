package queuebatch

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

func (lru *lru) access(key string) {
	if lru.h == nil && lru.t == nil {
		lru.h = &lruNode{data: key}
		lru.t = lru.h
		lru.km[key] = lru.h
		return
	}
	node, ok := lru.km[key]
	if !ok {
		// first time key access. So, create the node
		node = &lruNode{data: key, r: lru.h}
		lru.h.l = node
		lru.h = node
		lru.km[key] = lru.h
		return
	}
	if node == lru.h {
		return
	}
	node.l.r = node.r
	if node.r != nil {
		node.r.l = node.l
	} else {
		// remove the tail.
		if node.l != nil {
			node.l.r = nil
		}
		lru.t = node.l
	}
	lru.h.l = node
	node.l = nil
	node.r = lru.h
	lru.h = node

}

func (lru *lru) keyToEvict() string {
	if lru.t == nil {
		return ""
	}
	return lru.t.data
}

func (lru *lru) evict(key string) {
	if data, exists := lru.km[key]; exists {
		// if key is at head
		if data == lru.h {
			if data.r != nil {
				data.r.l = nil
			} else {
				// only one node
				lru.t = nil
			}
			lru.h = data.r
		} else if data == lru.t {
			if data.l != nil {
				data.l.r = nil
			}
			lru.t = data.l
		} else {
			data.l.r = data.r
			data.r.l = data.l
			data.l = nil
			data.r = nil
		}
		delete(lru.km, key)
	}
}
