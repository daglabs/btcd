package locks

import (
	"sync"
	"sync/atomic"
)

type waitGroup struct {
	counter  int64
	waitCond *sync.Cond
}

func newWaitGroup() *waitGroup {
	return &waitGroup{
		waitCond: sync.NewCond(&sync.Mutex{}),
	}
}

func (wg *waitGroup) add() {
	atomic.AddInt64(&wg.counter, 1)
}

func (wg *waitGroup) done() {
	counter := atomic.AddInt64(&wg.counter, -1)
	if counter < 0 {
		panic("negative values for wg.counter are not allowed. This was likely caused by calling done() before add()")
	}
	if atomic.LoadInt64(&wg.counter) == 0 {
		wg.waitCond.Broadcast()
	}
}

func (wg *waitGroup) wait() {
	wg.waitCond.L.Lock()
	defer wg.waitCond.L.Unlock()
	for atomic.LoadInt64(&wg.counter) != 0 {
		wg.waitCond.Wait()
	}
}
