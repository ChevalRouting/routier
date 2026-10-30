package flows

import (
	"time"
)

const (
	activeTimeout = 60 * time.Second
	idleTimeout   = 15 * time.Second
	maxCacheFlows = 65536
)

type entry struct {
	first   time.Time
	last    time.Time
	samples uint64
	bytes   uint64
}

type Record struct {
	Key     Key
	First   time.Time
	Last    time.Time
	Samples uint64
	Packets uint64
	Bytes   uint64
}

type cache struct {
	rate    uint64
	entries map[Key]*entry
}

func newCache(rate int) *cache {
	return &cache{rate: uint64(rate), entries: map[Key]*entry{}}
}

func (c *cache) add(s Sampled, now time.Time) bool {
	e := c.entries[s.Key]
	if e == nil {
		e = &entry{first: now}
		c.entries[s.Key] = e
	}

	e.last = now
	e.samples++
	e.bytes += uint64(s.Length)
	return len(c.entries) >= maxCacheFlows
}

func (c *cache) record(k Key, e *entry) Record {
	return Record{Key: k, First: e.first, Last: e.last, Samples: e.samples, Packets: e.samples * c.rate, Bytes: e.bytes * c.rate}
}

func (c *cache) expire(now time.Time, all bool) []Record {
	var out []Record
	for k, e := range c.entries {
		if all || now.Sub(e.last) >= idleTimeout || now.Sub(e.first) >= activeTimeout {
			out = append(out, c.record(k, e))
			delete(c.entries, k)
		}
	}

	return out
}
