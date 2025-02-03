package paxos

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicConsensus(t *testing.T) {
	numAcceptors := 3
	learner := NewLearner()
	learners := []*Learner{learner}

	var acceptors []*Acceptor
	for i := 0; i < numAcceptors; i++ {
		acceptor := NewAcceptor(learners)
		acceptors = append(acceptors, acceptor)
	}
	proposer := NewProposer(1, acceptors)
	expectedValue := "test-value"
	proposer.Propose(expectedValue)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, expectedValue, learner.GetValue())
}

func TestCompetingProposers(t *testing.T) {
	numAcceptors := 3
	learner := NewLearner()
	learners := []*Learner{learner}

	var acceptors []*Acceptor
	for i := 0; i < numAcceptors; i++ {
		acceptor := NewAcceptor(learners)
		acceptors = append(acceptors, acceptor)
	}

	proposer1 := NewProposer(1, acceptors)
	proposer2 := NewProposer(2, acceptors)

	value1 := "value1"
	value2 := "value2"

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		proposer1.Propose(value1)
	}()

	go func() {
		defer wg.Done()
		proposer2.Propose(value2)
	}()

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	learnedValue := learner.GetValue()
	assert.Contains(t, []string{value1, value2}, learnedValue)
}

func TestProposalIDOrdering(t *testing.T) {
	tests := []struct {
		p1       ProposalID
		p2       ProposalID
		expected bool
	}{
		{
			ProposalID{NodeID: 1, Sequence: 1},
			ProposalID{NodeID: 1, Sequence: 2},
			false,
		},
		{
			ProposalID{NodeID: 2, Sequence: 1},
			ProposalID{NodeID: 1, Sequence: 1},
			true,
		},
		{
			ProposalID{NodeID: 1, Sequence: 2},
			ProposalID{NodeID: 2, Sequence: 1},
			true,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.p1.GreaterThanOrEqual(tt.p2))
	}
}
