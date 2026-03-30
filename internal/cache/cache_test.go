package cache

import (
	"testing"
)

func TestCacheDisabled(t *testing.T) {
	c := New(0, "cfg")
	c.Put("k", "v")
	if _, ok := c.Get("k"); ok {
		t.Fatal("disabled cache should always miss")
	}
}

func TestCacheHitMiss(t *testing.T) {
	c := New(10, "cfg")

	// Miss
	if _, ok := c.Get("k1"); ok {
		t.Fatal("expected miss on empty cache")
	}

	// Put and hit
	c.Put("k1", "value1")
	v, ok := c.Get("k1")
	if !ok {
		t.Fatal("expected hit after put")
	}
	if v != "value1" {
		t.Fatalf("got %q, want %q", v, "value1")
	}

	hits, misses := c.Stats()
	if hits != 1 || misses != 1 {
		t.Fatalf("stats: hits=%d misses=%d, want 1/1", hits, misses)
	}
}

func TestCacheEviction(t *testing.T) {
	c := New(3, "cfg")
	c.Put("a", "1")
	c.Put("b", "2")
	c.Put("c", "3")
	// Cache full: [c, b, a]

	c.Put("d", "4") // evicts a (LRU)

	if _, ok := c.Get("a"); ok {
		t.Fatal("entry 'a' should have been evicted")
	}
	if v, ok := c.Get("d"); !ok || v != "4" {
		t.Fatalf("entry 'd' should be present, got ok=%v v=%q", ok, v)
	}
}

func TestCacheLRUOrder(t *testing.T) {
	c := New(3, "cfg")
	c.Put("a", "1")
	c.Put("b", "2")
	c.Put("c", "3")
	// [c, b, a]

	// Access 'a' to make it MRU
	c.Get("a")
	// [a, c, b]

	c.Put("d", "4") // evicts 'b' (now LRU)
	// [d, a, c]

	if _, ok := c.Get("b"); ok {
		t.Fatal("entry 'b' should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("entry 'a' should still be present (was accessed)")
	}
}

func TestCacheUpdateExisting(t *testing.T) {
	c := New(10, "cfg")
	c.Put("k", "v1")
	c.Put("k", "v2")

	v, ok := c.Get("k")
	if !ok || v != "v2" {
		t.Fatalf("expected updated value 'v2', got ok=%v v=%q", ok, v)
	}
}

func TestCacheKey(t *testing.T) {
	c := New(10, "cfg1")

	k1 := c.Key("hello", 1, 50)
	k2 := c.Key("hello", 1, 50)
	k3 := c.Key("hello", 2, 50)  // different cmdType
	k4 := c.Key("hello", 1, 100) // different maxLines
	k5 := c.Key("world", 1, 50)  // different content

	if k1 != k2 {
		t.Fatal("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Fatal("different cmdType should produce different key")
	}
	if k1 == k4 {
		t.Fatal("different maxLines should produce different key")
	}
	if k1 == k5 {
		t.Fatal("different content should produce different key")
	}

	// Different config hash
	c2 := New(10, "cfg2")
	k6 := c2.Key("hello", 1, 50)
	if k1 == k6 {
		t.Fatal("different config should produce different key")
	}
}

func TestConfigHash(t *testing.T) {
	h1 := ConfigHash("f1", "r1", "balanced", true)
	h2 := ConfigHash("f1", "r1", "balanced", true)
	h3 := ConfigHash("f1", "r1", "aggressive", true)

	if h1 != h2 {
		t.Fatal("same inputs should produce same hash")
	}
	if h1 == h3 {
		t.Fatal("different mode should produce different hash")
	}
}
