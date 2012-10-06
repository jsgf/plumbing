// Route messages from one channel over a set of other channels
package multipath

type MultipathMsg struct {
	seq uint
	payload *chan interface{}
}

func MultipathOut(in <-chan interface{}, out ...chan<- MultipathMsg) {
}

func MultipathIn(in ...<-chan MultipathMsg, out chan<- interface{}) {
}
