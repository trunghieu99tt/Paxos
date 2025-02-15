package paxos

import (
	"context"
	"log"
	"sync"
	"time"
)

type LearnerInterface interface {
	Run(ctx context.Context) (interface{}, bool)
	GetValue() interface{}
}

type Learner struct {
	mu       sync.RWMutex
	id       uint32
	network  INetwork
	accepted map[uint32]Message
	value    interface{}
}

func NewLearner(id uint32, network INetwork, acceptorIds []uint32) *Learner {
	accepted := make(map[uint32]Message, len(acceptorIds))
	for _, acceptorId := range acceptorIds {
		accepted[acceptorId] = Message{Type: DumbMessage}
	}
	l := &Learner{
		id:       id,
		network:  network,
		accepted: accepted,
	}
	return l
}

func (l *Learner) Run(ctx context.Context) (value interface{}, ok bool) {
	for l.value == nil {
		select {
		case <-time.After(LearnerTimeout):
			return nil, false
		default:
			accepted, err := l.network.Receive(ctx, l.id, 1*time.Second)
			if err != nil {
				continue
			}
			if accepted.Type != LearnMessage {
				continue
			}
			l.receiveAccepted(accepted)
			chosenValue, ok := l.chosen()
			if ok {
				l.mu.Lock()
				l.value = chosenValue
				l.mu.Unlock()
				return chosenValue, true
			}
		}
	}
	return nil, false
}

func (l *Learner) GetValue() interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.value
}

func (l *Learner) receiveAccepted(accepted Message) {
	if accepted.Value == nil {
		return
	}
	newAcceptedResponse := accepted.Value.(AcceptResponse)
	currLearnt := l.accepted[accepted.From]
	if currLearnt.Type == DumbMessage || newAcceptedResponse.Ok && newAcceptedResponse.Number > currLearnt.Value.(AcceptResponse).Number {
		log.Printf("learner: %d received a new accepted proposal %+v", l.id, accepted)
		l.accepted[accepted.From] = accepted
	}
}

func (l *Learner) majority() int {
	return len(l.accepted)/2 + 1
}

// A proposal is chosen when it has been accepted by a majority of the
// acceptors.
// The leader might choose multiple proposals when it learns multiple times,
// but we guarantee that all chosen proposals have the same value.
func (l *Learner) chosen() (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	counts := make(map[int64]int)
	acceptedCnt := make(map[int64]Message)

	for _, accepted := range l.accepted {
		if accepted.Type != LearnMessage {
			continue
		}
		accept := accepted.Value.(AcceptResponse)
		counts[accept.Number]++
		acceptedCnt[accept.Number] = accepted
	}

	for n, count := range counts {
		if count >= l.majority() {
			return acceptedCnt[n].Value, true
		}
	}
	return nil, false
}
