package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	WaitForNextElementChanCapacity           = 1000
	dequeueOrWaitForNextElementInvokeGapTime = 10
	QueueErrorCodeEmptyQueue                 = "empty-queue"
	QueueErrorCodeLockedQueue                = "locked-queue"
	QueueErrorCodeIndexOutOfBounds           = "index-out-of-bounds"
)

type QueueError struct {
	code    string
	message string
}

func NewQueueError(code string, message string) *QueueError {
	return &QueueError{
		code:    code,
		message: message,
	}
}

func (st *QueueError) Error() string {
	return st.message
}

func (st *QueueError) Code() string {
	return st.code
}

// FIFO (First In First Out) concurrent queue
type FIFO struct {
	slice       []interface{}
	rwmutex     sync.RWMutex
	lockRWmutex sync.RWMutex
	isLocked    bool
	// queue for watchers that will wait for next elements (if queue is empty at DequeueOrWaitForNextElement execution )
	waitForNextElementChan chan chan interface{}
	// queue to unlock consumers that were locked when queue was empty (during DequeueOrWaitForNextElement execution)
	unlockDequeueOrWaitForNextElementChan chan struct{}
}

// NewFIFO returns a new FIFO concurrent queue
func NewFIFO() *FIFO {
	ret := &FIFO{}
	ret.initialize()

	return ret
}

func (st *FIFO) initialize() {
	st.slice = make([]interface{}, 0)
	st.waitForNextElementChan = make(chan chan interface{}, WaitForNextElementChanCapacity)
	st.unlockDequeueOrWaitForNextElementChan = make(chan struct{}, WaitForNextElementChanCapacity)
}

// Enqueue enqueues an element. Returns error if queue is locked.
func (st *FIFO) Enqueue(value interface{}) error {
	if st.isLocked {
		return NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	// let consumers (DequeueOrWaitForNextElement) know there is a new element
	select {
	case st.unlockDequeueOrWaitForNextElementChan <- struct{}{}:
	default:
		// message could not be sent
	}

	// check if there is a listener waiting for the next element (this element)
	select {
	case listener := <-st.waitForNextElementChan:
		// send the element through the listener's channel instead of enqueue it
		select {
		case listener <- value:
		default:
			// enqueue if listener is not ready

			// lock the object to enqueue the element into the slice
			st.rwmutex.Lock()
			// enqueue the element
			st.slice = append(st.slice, value)
			defer st.rwmutex.Unlock()
		}

	default:
		// lock the object to enqueue the element into the slice
		st.rwmutex.Lock()
		// enqueue the element
		st.slice = append(st.slice, value)
		defer st.rwmutex.Unlock()
	}

	return nil
}

// Dequeue dequeues an element. Returns error if queue is locked or empty.
func (st *FIFO) Dequeue() (interface{}, error) {
	if st.isLocked {
		return nil, NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	st.rwmutex.Lock()
	defer st.rwmutex.Unlock()

	if len(st.slice) == 0 {
		return nil, NewQueueError(QueueErrorCodeEmptyQueue, "empty queue")
	}

	elementToReturn := st.slice[0]
	st.slice = st.slice[1:]

	return elementToReturn, nil
}

// DequeueOrWaitForNextElement dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
// Multiple calls to DequeueOrWaitForNextElement() would enqueue multiple "listeners" for future enqueued elements.
func (st *FIFO) DequeueOrWaitForNextElement() (interface{}, error) {
	return st.DequeueOrWaitForNextElementContext(context.Background())
}

// DequeueOrWaitForNextElementContext dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
// Multiple calls to DequeueOrWaitForNextElementContext() would enqueue multiple "listeners" for future enqueued elements.
// When the passed context expires this function exits and returns the context' error
func (st *FIFO) DequeueOrWaitForNextElementContext(ctx context.Context) (interface{}, error) {
	for {
		if st.isLocked {
			return nil, NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
		}

		// get the slice's len
		st.rwmutex.Lock()
		length := len(st.slice)
		st.rwmutex.Unlock()

		if length == 0 {
			// channel to wait for next enqueued element
			waitChan := make(chan interface{})

			select {
			// enqueue a watcher into the watchForNextElementChannel to wait for the next element
			case st.waitForNextElementChan <- waitChan:

				// n
				for {
					// re-checks every i milliseconds (top: 10 times) ... the following verifies if an item was enqueued
					// around the same time DequeueOrWaitForNextElementContext was invoked, meaning the waitChan wasn't yet sent over
					// st.waitForNextElementChan
					for i := 0; i < dequeueOrWaitForNextElementInvokeGapTime; i++ {
						select {
						case <-ctx.Done():
							return nil, ctx.Err()
						case dequeuedItem := <-waitChan:
							return dequeuedItem, nil
						case <-time.After(time.Millisecond * time.Duration(i)):
							if dequeuedItem, err := st.Dequeue(); err == nil {
								return dequeuedItem, nil
							}
						}
					}

					// return the next enqueued element, if any
					select {
					// new enqueued element, no need to keep waiting
					case <-st.unlockDequeueOrWaitForNextElementChan:
						// check if we got a new element just after we got <-st.unlockDequeueOrWaitForNextElementChan
						select {
						case item := <-waitChan:
							return item, nil
						default:
						}
						// go back to: for loop
						continue

					case item := <-waitChan:
						return item, nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					// n
				}
			default:
				// too many watchers (waitForNextElementChanCapacity) enqueued waiting for next elements
				return nil, NewQueueError(QueueErrorCodeEmptyQueue, "empty queue and can't wait for next element because there are too many DequeueOrWaitForNextElement() waiting")
			}
		}

		st.rwmutex.Lock()

		// verify that at least 1 item resides on the queue
		if len(st.slice) == 0 {
			st.rwmutex.Unlock()
			continue
		}
		elementToReturn := st.slice[0]
		st.slice = st.slice[1:]

		st.rwmutex.Unlock()
		return elementToReturn, nil
	}
}

// Get returns an element's value and keeps the element at the queue
func (st *FIFO) Get(index int) (interface{}, error) {
	if st.isLocked {
		return nil, NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	st.rwmutex.RLock()
	defer st.rwmutex.RUnlock()

	if len(st.slice) <= index {
		return nil, NewQueueError(QueueErrorCodeIndexOutOfBounds, fmt.Sprintf("index out of bounds: %v", index))
	}

	return st.slice[index], nil
}

// Remove removes an element from the queue
func (st *FIFO) Remove(index int) error {
	if st.isLocked {
		return NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	st.rwmutex.Lock()
	defer st.rwmutex.Unlock()

	if len(st.slice) <= index {
		return NewQueueError(QueueErrorCodeIndexOutOfBounds, fmt.Sprintf("index out of bounds: %v", index))
	}

	// remove the element
	st.slice = append(st.slice[:index], st.slice[index+1:]...)

	return nil
}

// GetLen returns the number of enqueued elements
func (st *FIFO) GetLen() int {
	st.rwmutex.RLock()
	defer st.rwmutex.RUnlock()

	return len(st.slice)
}

// GetCap returns the queue's capacity
func (st *FIFO) GetCap() int {
	st.rwmutex.RLock()
	defer st.rwmutex.RUnlock()

	return cap(st.slice)
}

// Lock // Locks the queue. No enqueue/dequeue operations will be allowed after this point.
func (st *FIFO) Lock() {
	st.lockRWmutex.Lock()
	defer st.lockRWmutex.Unlock()

	st.isLocked = true
}

// Unlock unlocks the queue
func (st *FIFO) Unlock() {
	st.lockRWmutex.Lock()
	defer st.lockRWmutex.Unlock()

	st.isLocked = false
}

// IsLocked returns true whether the queue is locked
func (st *FIFO) IsLocked() bool {
	st.lockRWmutex.RLock()
	defer st.lockRWmutex.RUnlock()

	return st.isLocked
}
