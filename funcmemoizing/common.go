package funcmemoizing

import "io"

type GenFunc func(key string) (interface{}, error)

type result struct {
	value interface{}
	err   error
}

type entry struct {
	res    result
	ready chan struct{}
}

type MemoCache interface {
	io.Closer
	Get(key string) (interface{}, error)
}