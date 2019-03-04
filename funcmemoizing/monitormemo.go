package funcmemoizing

type request struct {
	key      string
	response chan<- result
}

type MonitorMemo struct {
	requests chan request
}

func (memo *MonitorMemo) Get(key string) (interface{}, error)  {
	response := make(chan result)
	memo.requests <- request{key:key, response:response}
	res := <-response
	return res.value, res.err
}

func (memo *MonitorMemo) Close() error {
	close(memo.requests)
	return nil
}

func (memo *MonitorMemo) serve(genFunc GenFunc)  {
	cache := make(map[string]*entry)
	for request := range memo.requests{
		e := cache[request.key]
		if e == nil {
			e = &entry{ready: make(chan struct{})}
			cache[request.key] = e
			go e.call(genFunc, request.key)
		}
		go e.deliver(request.response)
	}
}

func (e *entry) call(genFunc GenFunc, key string)  {
	e.res.value, e.res.err = genFunc(key)
	close(e.ready)
}

func (e *entry) deliver(response chan<- result)  {
	<- e.ready
	response <- e.res
}

func NewMonitorMemo(genFunc GenFunc) MemoCache  {
	memo := &MonitorMemo{requests: make(chan request)}
	go memo.serve(genFunc)
	return memo
}