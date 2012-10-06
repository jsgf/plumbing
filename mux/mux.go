// mux allows channels to be multiplexed into and demultiplexed from a single channel
package mux

import "fmt"

// Chanid is an identifier for a specific channel.
type Chanid struct {
	chanid uint
}

// A Bundle is a set of channels multiplexed together.
type Bundle struct {
	chanid  uint
	payload *interface{} // payload == nil means close
}

type muxctl struct {
	ret chan uint
	ch  <-chan interface{}
}

type demuxctl struct {
	chanid uint
	ch     chan interface{}
	ret    chan (<-chan interface{})
}

// Demultiplexer for a specific Bundle.
type Demuxer struct {
	ctl chan<- demuxctl
}

// Multiplexer for a specific Bundle.
type Muxer struct {
	ctl chan<- muxctl
}

func demuxproc(ctlch <-chan demuxctl, muxed <-chan Bundle) {
	idmap := make(map[uint]chan interface{})
	defer func() {
		for _, ch := range idmap {
			close(ch)
		}
	}()

	for {
		select {
		case ctl, ok := <-ctlch:
			if !ok {
				return
			}

			ch, ok := idmap[ctl.chanid]
			if !ok {
				ch = ctl.ch
				idmap[ctl.chanid] = ch
			} else {
				close(ctl.ch)
			}

			ctl.ret <- ch

		case msg, ok := <-muxed:
			if !ok {
				return
			}

			if ch, ok := idmap[msg.chanid]; ok {
				if msg.payload == nil {
					close(ch)
					delete(idmap, msg.chanid)
				} else {
					ch <- *msg.payload
				}
			} else {
				// This isn't very nice.  Alternative
				// would be to proactively install a
				// channel in the map and start
				// sending to it in the hope that
				// someone registers a demux for it,
				// but that leaves the risk of
				// blocking on the send and
				// deadlocking (since we wouldn't
				// handle ctl messages either).
				fmt.Println("Dropping id", msg.chanid, msg.payload)
			}
		}
	}
}

// Create a demultiplexer for a Bundle.
func MakeDemuxer(ch <-chan Bundle) *Demuxer {
	ctl := make(chan demuxctl)
	ret := &Demuxer{ctl}

	go demuxproc(ctl, ch)

	return ret
}

// Demultiplex a channel with the given Chanid out of a Bundle.
func (demux *Demuxer) Demux(id Chanid, ch chan interface{}) <-chan interface{} {
	if ch == nil {
		ch = make(chan interface{})
	}

	ret := make(chan (<-chan interface{}))
	demux.ctl <- demuxctl{id.chanid, ch, ret}

	return <-ret
}

func muxproc(ctlch <-chan muxctl, muxed chan<- Bundle) {
	curid := uint(0)
	chmap := make(map[<-chan interface{}]uint)

	finished := make(chan (<-chan interface{}))

	for {
		select {
		case ch := <-finished:
			delete(chmap, ch)

		case ctl, ok := <-ctlch:
			id, ok := chmap[ctl.ch]
			if !ok {
				id = curid
				curid++

				chmap[ctl.ch] = id

				go func(id uint, ch <-chan interface{}) {
					defer func() { finished <- ch }()

					for msg := range ch {
						muxed <- Bundle{id, &msg}
					}
					muxed <- Bundle{id, nil}
				}(id, ctl.ch)
			}
			ctl.ret <- id
		}
	}
}

// Make a multiplexer for a Bundle channel.
func MakeMuxer(ch chan<- Bundle) *Muxer {
	ctl := make(chan muxctl)
	ret := &Muxer{ctl}

	go muxproc(ctl, ch)

	return ret
}

// Mux multiplexes a channel into a multiplexer, returning the Chanid.
func (mux *Muxer) Mux(ch <-chan interface{}) Chanid {
	ret := make(chan uint)
	mux.ctl <- muxctl{ret, ch}

	return Chanid{<-ret}
}
