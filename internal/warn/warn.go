// Package warn aggregates conversion warnings.
package warn

import (
	"sort"
	"sync"
)

type Warning struct {
	Kind, Key, Message string
	Count              int
}
type Collector struct {
	mu      sync.Mutex
	entries map[[2]string]Warning
}

func (c *Collector) Add(kind, key, message string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[[2]string]Warning)
	}
	k := [2]string{kind, key}
	v := c.entries[k]
	if v.Count == 0 {
		v = Warning{Kind: kind, Key: key, Message: message}
	}
	v.Count++
	c.entries[k] = v
}
func (c *Collector) Sorted() []Warning {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Warning, 0, len(c.entries))
	for _, v := range c.entries {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return out
}
