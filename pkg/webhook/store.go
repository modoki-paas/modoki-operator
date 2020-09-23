package webhook

import (
	"log"
	"sync"
)

// Filter should return the event is catched or not
type Filter = func(event string) bool

// Operation is called when Filter returned true
// Each Operation will not called twice or more at the same time
type Operation = func(event string, payload []byte)

var (
	lock       sync.RWMutex
	registered = map[string]*funcPair{}
)

type funcPair struct {
	lock sync.Mutex
	f    Filter
	op   Operation
}

// Register registers Filter/Operation in the store
func Register(name string, f Filter, op Operation) {
	lock.Lock()
	defer lock.Unlock()
	registered[name] = &funcPair{
		f:  f,
		op: op,
	}
}

func call(event string, payload []byte) {
	lock.RLock()
	defer lock.RUnlock()
	for name, fp := range registered {
		if fp.f(event) {
			log.Println(name, "found")
			go func() {
				fp.lock.Lock()
				defer fp.lock.Unlock()
				fp.op(event, payload)
			}()
		}
	}
}
