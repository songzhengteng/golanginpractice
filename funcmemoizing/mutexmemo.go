package funcmemoizing

import "sync"

type MutexMemo struct {
	genFuc GenFunc

	mu    sync.Mutex // protect cache
	cache map[string]*entry
}

func (memo *MutexMemo) Get(key string) (interface{}, error) {
	memo.mu.Lock()
	e := memo.cache[key];
	if e == nil {
		e = &entry{ready: make(chan struct{})}
		memo.cache[key] = e
		memo.mu.Unlock()
		e.res.value, e.res.err = memo.genFuc(key)
		close(e.ready)
	} else {
		memo.mu.Unlock()
		<-e.ready // block until first GenFunc called
	}
	return e.res.value, e.res.err
}

func (memo *MutexMemo) Close() error  {
	return nil
}

func New(f GenFunc) MemoCache {
	return &MutexMemo{genFuc: f, cache: make(map[string]*entry)}
}


