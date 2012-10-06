// Implement unbounded channel buffer
package buffer

import (
	"container/list"
	"sync"
)

// Buffer receives the input channel until it is closed, and buffers
// output indefinitely until it is sent to the output channel.
func Buffer(in <-chan interface{}) (out chan<- interface{}) {
	out = make(chan interface{}, 1)

	cv := sync.NewCond(&sync.Mutex{})
	l := list.New()

	go func() {
		cv.L.Lock()

		sig := false
		empty := true

		for {
			var ok bool
			var v interface{}

			select {
			case v,ok = <-in:
				// Non-blocking read; don't touch lock

			default:
				if sig {
					cv.Signal()
				}

				cv.L.Unlock()
				v,ok = <- in
				cv.L.Lock()

				empty = l.Front() == nil
			}

			if !ok {
				break
			}

			sig = empty
			l.PushBack(&v)
		}

		l.PushBack(nil)
		cv.L.Unlock()
	}()

	go func() {
		cv.L.Lock()
		for {
			e := l.Front()
			if e == nil {
				cv.Wait()
				continue
			}
			v := e.Value.(*interface{})

			if v == nil {
				close(out)
				break
			}

			select {
			case out <- *v:
				// try non-blocking while holding lock

			default:
				// would block, release lock to send
				cv.L.Unlock()
				out <- *v
				cv.L.Lock()
			}
			l.Remove(e)
		}
		cv.L.Unlock()
	}()

	return
}
