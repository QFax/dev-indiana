package services

import ()

type APIKeyResult struct {
	APIKey string
	Error  error
}

type Request struct {
	APIKeyChan chan APIKeyResult
}

type RequestQueue struct {
	queue chan *Request
}

func NewRequestQueue() *RequestQueue {
	return &RequestQueue{
		queue: make(chan *Request, 100),
	}
}

func (q *RequestQueue) Add(req *Request) {
	q.queue <- req
}

func (q *RequestQueue) Get() *Request {
	return <-q.queue
}