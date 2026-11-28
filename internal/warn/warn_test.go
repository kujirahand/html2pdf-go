package warn

import (
	"sync"
	"testing"
)

func TestCollector(t *testing.T) {
	var nilCollector *Collector
	nilCollector.Add("x", "y", "z")
	if nilCollector.Sorted() != nil {
		t.Fatal("nil collector")
	}
	c := new(Collector)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Add("b", "z", "message") }()
	}
	wg.Wait()
	c.Add("a", "z", "first")
	c.Add("a", "a", "second")
	got := c.Sorted()
	if len(got) != 3 || got[0].Key != "a" || got[1].Kind != "a" || got[2].Count != 20 {
		t.Fatalf("%+v", got)
	}
}
