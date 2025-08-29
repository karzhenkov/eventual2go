package eventual2go

import (
	"fmt"
	"sync"
	"time"
)

// Future is thread-safe struct that can be completed with arbitrary data or failed with an error. Handler functions can
// be registered for both events and get invoked after completion..
type Future[T any] struct {
	m         *sync.RWMutex
	fcs       []CompletionHandler[T]
	fces      []ErrorHandler
	completed bool
	result    T
	err       error
}

// Creates a new future.
func newFuture[T any]() (F *Future[T]) {
	F = &Future[T]{
		m:         new(sync.RWMutex),
		fcs:       []CompletionHandler[T]{},
		fces:      []ErrorHandler{},
		completed: false,
	}
	return
}

// Completes the future with the given data and triggers al registered completion handlers. Returns false if the future is already
// complete.
func (f *Future[T]) tryComplete(d T) bool {
	defer LockGuard(f.m)()
	if f.completed {
		return false
	}
	f.result = d
	for _, fc := range f.fcs {
		go fc(d)
	}
	f.completed = true
	return true
}

// Completes the future with the given data and triggers al registered completion handlers. Panics if the future is already
// complete.
func (f *Future[T]) complete(d T) {
	if !f.tryComplete(d) {
		panic(fmt.Sprint("Completed complete future with", d))
	}
}

// Completed returns the completion state.
func (f *Future[T]) Completed() bool {
	defer RLockGuard(f.m)()
	return f.completed
}

// Completes the future with the given error and triggers al registered error handlers. Returns false if the future is already
// complete.
func (f *Future[T]) tryCompleteError(err error) bool {
	defer LockGuard(f.m)()
	if f.completed {
		return false
	}
	f.err = err
	for _, fce := range f.fces {
		fce(f.err)
	}
	f.completed = true
	return true
}

// Completes the future with the given error and triggers al registered error handlers. Panics if the future is already
// complete.
func (f *Future[T]) completeError(err error) {
	if !f.tryCompleteError(err) {
		panic(fmt.Sprint("Errorcompleted complete future with", err))
	}
}

// Then registers a completion handler. If the future is already complete, the handler gets executed immediately.
// Returns a future that gets completed with result of the handler.
func (f *Future[T]) Then(ch CompletionHandler[T]) {

	f.m.Lock()
	if !f.completed {
		f.fcs = append(f.fcs, ch)
	} else if f.err == nil {
		defer ch(f.result)
	}
	f.m.Unlock()
}

// WaitUntilComplete blocks until the future is complete.
func (f *Future[T]) WaitUntilComplete() {
	<-f.AsChan()
}

// WaitUntilTimeout blocks until the future is complete or the timeout is reached.
func (f *Future[T]) WaitUntilTimeout(timeout time.Duration) (complete bool) {
	if !f.Completed() {
		select {
		case <-f.AsChan():
		case <-time.After(timeout):
		}
	}
	complete = f.Completed()
	return
}

// Result returns the result of the future, nil if called after error completion.
// Should be called only after Complete() returns true.
func (f *Future[T]) Result() T {
	return f.result
}

// ErrResult returns the resulting error of the future, nil if called after non-error completion.
// Should be called only after Complete() returns true.
func (f *Future[T]) ErrResult() error {
	return f.err
}

// Outcome returns outcome of the future, nil if called before completion
func (f *Future[T]) Outcome() *Outcome[T] {
	if !f.Completed() {
		return nil
	}

	return &Outcome[T]{
		Result: f.result,
		Error:  f.err,
	}
}

// Err registers an error handler. If the future is already completed with an error, the handler gets executed
// immediately.
// Returns a future that either gets completed with result of the handler or error completed with the error from handler,
// if not nil.
func (f *Future[T]) Err(eh ErrorHandler) {
	f.m.Lock()

	if !f.completed {
		f.fces = append(f.fces, eh)
	} else if f.err != nil {
		defer eh(f.err)
	}
	f.m.Unlock()
}

// AsChan returns a channel which either will receive the result on completion or gets closed on error completion of the Future.
func (f *Future[T]) AsChan() <-chan T {
	c := make(chan T, 1)
	f.Then(func(d T) {
		c <- d
		close(c)
	})
	f.Err(func(error) {
		close(c)
	})
	return c
}

// AsErrChan returns a channel which either will receive the error on error completion or gets closed on completion of the Future.
func (f *Future[T]) AsErrChan() <-chan error {
	c := make(chan error, 1)
	f.Then(func(T) {
		close(c)
	})
	f.Err(func(e error) {
		c <- e
		close(c)
	})
	return c
}
