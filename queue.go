package main

import (
	"log"
	"sync"
)

// Create basic structure of a Request
// Contains the following information:
// Value: Contains the Address of where the request is coming from
type Request struct {
	Value string
}

/**
* NewQueue returns a new queue with the size passed through param
* @param size the size to intilialize this queue
* @return *Queue a new Queue structure with the provided size
**/
func NewQueue(size int) *Queue {
	log.Printf("creating queue of size %d\n", size)
	return &Queue{
		requests: make([]*Request, size),
		size:     size,
		head:     0,
		tail:     0,
		count:    0,
	}
}

// Create a Queue structure that is FIFO data structure
// Contains the following information:
// size: Stores the size of the Queue
// head: Stores the location of the front of the Queue
// tail: Stores the location of the back of the Queue
// count: Stores the number of Requests within the Queue
// muxForPush: Provides lock/unlock functionality for Push since this Queue will be accessed by multiple threads (we don't want several threads modifying variables at the same time)
// muxForPop: Provides lock/unlock functionality for Pop since this Queue will be accessed by multiple threads (we don't want several threads modifying variables at the same time)
type Queue struct {
	requests   []*Request
	size       int
	head       int
	tail       int
	count      int
	muxForPush sync.Mutex
	muxForPop  sync.Mutex
}

/**
* Push a Request onto the Queue
* @param Request the new request that will be pushed onto the Queue
* if the Queue is full, then function will halt until it becomes empty
**/
func (q *Queue) Push(n *Request) {

	//various threads can access these values, so perform lock functionality here
	// Lock so only one goroutine at a time can access these variables
	q.muxForPush.Lock()
	if q.size <= q.count {
		for q.count > 0 {
			//log.Printf("======== QUEUE is full.... ==========")
		}
	}

	q.requests[q.tail] = n
	q.tail = (q.tail + 1) % len(q.requests)
	q.count++
	q.muxForPush.Unlock()

}

/**
* Pop a Request from the Queue
* @return request the Request that has been popped from the Queue
**/
func (q *Queue) Pop() *Request {

	//various threads can access these values, so perform lock functionality here
	// Lock so only one goroutine at a time can access these variables
	q.muxForPop.Lock()
	if q.count == 0 {
		return nil
	}
	request := q.requests[q.head]
	q.head = (q.head + 1) % len(q.requests)
	q.count--
	q.muxForPop.Unlock()

	return request
}
