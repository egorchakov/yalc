package main

import (
	"container/list"
	"net/url"
)

type urlState int

const (
	untrackedState urlState = iota
	queuedState
	processingState
	processedState
	errorState
)

func (s urlState) String() string {
	switch s {
	case untrackedState:
		return "untracked"
	case queuedState:
		return "queued"
	case processingState:
		return "processing"
	case processedState:
		return "processed"
	case errorState:
		return "error"
	default:
		return "unknown"
	}
}

type transition struct {
	prev urlState
	next urlState
}

type counters struct {
	processing uint
	error      uint
}

type manager struct {
	seed *url.URL

	queue    *list.List
	state    map[string]urlState
	counters counters

	inCh     <-chan *page
	errCh    <-chan *page
	outCh    chan<- *page
	resultCh chan<- Result

	result Result
}

func newManager(seed *url.URL, inCh <-chan *page, outCh chan<- *page, errCh <-chan *page, resultCh chan<- Result) *manager {
	return &manager{
		seed:     seed,
		queue:    list.New(),
		state:    make(map[string]urlState),
		counters: counters{},
		result:   Result{links: make(map[string][]string)},
		inCh:     inCh,
		outCh:    outCh,
		errCh:    errCh,
		resultCh: resultCh,
	}
}

func (m *manager) Run() {
	var localOutCh chan<- *page
	var queueFront *list.Element

	m.enqueue(m.seed)

	for {
		if m.queue.Len() > 0 {
			localOutCh = m.outCh
			queueFront = m.queue.Front()
		} else {
			if m.counters.processing == 0 {
				m.result.errorCount = m.counters.error
				m.resultCh <- m.result

				return
			}

			localOutCh = nil
		}

		select {
		case page := <-m.errCh:
			m.setState(*page.url, errorState)
			log.Debugw("error", "url", page.url)

		case page := <-m.inCh:
			children := make([]string, 0, len(page.children))

			for _, child := range page.children {
				children = append(children, child.String())

				if m.getState(*child) == untrackedState {
					m.enqueue(child)
				}
			}

			m.result.links[page.url.String()] = children
			m.setState(*page.url, processedState)

			log.Infow("processed", "url", page.url, "children", len(page.children))

		case localOutCh <- &page{url: queueFront.Value.(*url.URL)}:
			m.dequeue(queueFront)
			log.Debugw("send", "url", queueFront.Value.(*url.URL).String())
		}
	}
}

func (m *manager) setState(u url.URL, state urlState) {
	t := transition{m.getState(u), state}

	switch t {
	case transition{untrackedState, queuedState}:

	case transition{queuedState, processingState}:
		m.counters.processing++

	case transition{processingState, processedState}:
		m.counters.processing--

	case transition{processingState, errorState}:
		m.counters.processing--
		m.counters.error++

	default:
		log.Panicw("invalid state transition", "url", u.String(), "prev", t.prev, "next", t.next)
	}

	// Ignore the scheme (http vs https) for tracking
	u.Scheme = ""
	m.state[u.String()] = state
}

func (m *manager) getState(u url.URL) urlState {
	u.Scheme = ""
	if val, ok := m.state[u.String()]; ok {
		return val
	}

	return untrackedState
}

func (m *manager) enqueue(u *url.URL) {
	m.queue.PushBack(u)
	m.setState(*u, queuedState)
}

func (m *manager) dequeue(el *list.Element) {
	u := el.Value.(*url.URL)
	m.queue.Remove(el)
	m.setState(*u, processingState)
}
