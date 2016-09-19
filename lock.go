package locky

import (
	"fmt"
)

type lockreq struct {
	ch   chan bool
	job  int // 0 lock , 1 unlock, 2 change
	path string
	p2   string
}

type reqLink struct {
	c    lockreq
	l, r *reqLink
}

func sContains(a, b string) bool {
	if len(a) <= len(b) {
		return (a == b[:len(a)])
	}
	return (b == a[:len(b)])
}
func sExact(a, b string) bool {
	return a == b
}

func loop(ch chan lockreq, compFunc func(string, string) bool) {
	queue := &reqLink{lockreq{}, nil, nil}
	qlast := queue
	live := make([]string, 0)
	lLen := 0
	qlen := 0
	fmt.Printf("Queue Len : %d\n", qlen)

	addLive := func(s string) {
		if lLen == len(live) {
			live = append(live, s)
			lLen++
			return
		}

		live[lLen] = s
		lLen++

	}
	remLive := func(i int) {
		live[i] = live[lLen-1]
		lLen--
		live[lLen] = ""
	}

	release := func(n int) {
		fcount := 0
		for curr := queue.r; curr != nil; curr = curr.r {

			found := false
			for i := 0; i < lLen; i++ {
				if compFunc(curr.c.path, live[i]) {
					found = true
					break
				}
			}
			if !found {
				addLive(curr.c.path)
				if curr.r == nil {
					qlast = curr.l
				} else {
					curr.r.l = curr.l
				}
				curr.l.r = curr.r
				qlen--
				fmt.Printf("Queue Len : %d\n", qlen)
				fcount++
				curr.c.ch <- true
				if fcount >= n {
					return
				}
			}
		}

	} //func release

	for req := range ch {
		switch req.job {
		case 0: //Lock
			match := false
			for i := 0; i < lLen; i++ {
				p := live[i]
				if compFunc(req.path, p) {
					match = true
					qlen++

					qlast.r = &reqLink{req, qlast, nil}
					qlast = qlast.r
					break
				}
			}
			if !match {
				addLive(req.path)
				req.ch <- true
			}

		case 1: // Unlock
			found := false
			for i := 0; i < lLen; i++ {
				if live[i] == req.path {
					remLive(i)
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("Unlock Not Found \"%v\"", req.path)
			}
			req.ch <- true
			release(1)

		case 2: // Dig
			fmt.Println("Digging")
			for i := 0; i < lLen; i++ {
				if live[i] == req.path {
					live[i] = req.p2
					break
				}
			}
			req.ch <- true
			release(1)
		default:
			req.ch <- false
		}

	}
}

type Locker chan lockreq

func BeginLoop() Locker {
	ch := make(chan lockreq)
	go loop(ch, sContains)
	return ch
}

func BeginSimple() Locker {
	ch := make(chan lockreq)
	go loop(ch, sExact)
	return ch
}

func (l Locker) Lock(p string) {
	gchan := make(chan bool)
	l <- lockreq{gchan, 0, p, ""}
	_ = <-gchan
}

func (l Locker) Unlock(p string) {
	gchan := make(chan bool)
	l <- lockreq{gchan, 1, p, ""}
	_ = <-gchan
}

// Dig will not check on grabbing a lock, q should be a smaller lock, contained by p.  That is: everything locked by q, will also lock with p
func (l Locker) Dig(p, q string) {
	gchan := make(chan bool)
	l <- lockreq{gchan, 2, p, q}
	_ = <-gchan
}
