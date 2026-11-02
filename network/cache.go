package network

import (
	"container/list"
	"sync"
	"time"
)

type cacheItem struct {
	key       string
	body      []byte
	timestamp time.Time
}

type lruCache struct {
	capacity  int
	items     map[string]*list.Element
	evictList *list.List
	mu        sync.Mutex
	ttl       time.Duration
}

func newLRUCache(capacity int, ttl time.Duration) *lruCache {
	return &lruCache{
		capacity:  capacity,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
		ttl:       ttl,
	}
}

func (c *lruCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		item := elem.Value.(*cacheItem)
		if time.Since(item.timestamp) > c.ttl {
			c.evictList.Remove(elem)
			delete(c.items, key)
			return nil, false
		}
		c.evictList.MoveToFront(elem)
		return item.body, true
	}
	return nil, false
}

func (c *lruCache) put(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.items[key]; exists {
		c.evictList.MoveToFront(elem)
		item := elem.Value.(*cacheItem)
		item.body = body
		item.timestamp = time.Now()
		return
	}

	if c.evictList.Len() >= c.capacity {
		oldest := c.evictList.Back()
		if oldest != nil {
			c.evictList.Remove(oldest)
			item := oldest.Value.(*cacheItem)
			delete(c.items, item.key)
		}
	}

	item := &cacheItem{key: key, body: body, timestamp: time.Now()}
	elem := c.evictList.PushFront(item)
	c.items[key] = elem
}

func (c *lruCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.evictList.Init()
}
