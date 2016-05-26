package main

import (
	"sync"
	"testing"
)

func TestInc(t *testing.T) {

	s := NewStats()

	loop := 20
	inc := 5

	var wg sync.WaitGroup
	wg.Add(loop)
	for i := 0; i < loop; i++ {
		go func() {
			defer wg.Done()
			for i := 0; i < inc; i++ {
				s.Inc(Consume)
			}
		}()
	}

	wg.Wait()

	expect := loop * inc
	if s.Consume != uint64(expect) {
		t.Fatalf("expect %d to be eq %d", s.Consume, expect)
	}
}
