package paxos

import "sync"

type Learner struct {
	mu    sync.Mutex
	value interface{}
	ch    chan interface{}
}

func NewLearner() *Learner {
	l := &Learner{ch: make(chan interface{}, 1)}
	go l.run()
	return l
}

func (l *Learner) Learn(value interface{}) {
	l.ch <- value
}

func (l *Learner) run() {
	for val := range l.ch {
		l.mu.Lock()
		l.value = val
		l.mu.Unlock()
	}
}

func (l *Learner) GetValue() interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.value
}
