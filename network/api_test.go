package network

import (
	"testing"
	"time"
)

func TestPtrInt(t *testing.T) {
	val := 42
	if ptrInt(&val) != 42 {
		t.Errorf("ptrInt(&val) = %d, expected 42", ptrInt(&val))
	}
	if ptrInt(nil) != 0 {
		t.Errorf("ptrInt(nil) = %d, expected 0", ptrInt(nil))
	}
}

func TestLRUCache(t *testing.T) {
	cache := newLRUCache(2, time.Second)

	cache.put("k1", []byte("v1"))
	cache.put("k2", []byte("v2"))

	if val, ok := cache.get("k1"); !ok || string(val) != "v1" {
		t.Errorf("expected k1='v1', got %q, ok=%v", string(val), ok)
	}

	// Add k3, should evict k2 (least recently used)
	cache.put("k3", []byte("v3"))

	if _, ok := cache.get("k2"); ok {
		t.Errorf("expected k2 to be evicted")
	}

	if val, ok := cache.get("k3"); !ok || string(val) != "v3" {
		t.Errorf("expected k3='v3', got %q, ok=%v", string(val), ok)
	}

	cache.clear()
	if _, ok := cache.get("k1"); ok {
		t.Errorf("expected cache to be cleared")
	}
}
