package paxos

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Proposer struct {
	mu        sync.Mutex
	acceptors []*Acceptor
	quorum    int
	nodeID    int
	sequence  int
}

func NewProposer(nodeID int, acceptors []*Acceptor) *Proposer {
	return &Proposer{
		acceptors: acceptors,
		quorum:    len(acceptors)/2 + 1,
		nodeID:    nodeID,
		sequence:  0,
	}
}

func (p *Proposer) newProposalID() ProposalID {
	p.sequence++
	return ProposalID{NodeID: p.nodeID, Sequence: p.sequence}
}

func (p *Proposer) Propose(value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	backoff := 50 * time.Millisecond
	for {
		proposalID := p.newProposalID()
		highestAcceptedValue, ok := p.preparePhase(proposalID)
		if !ok {
			fmt.Println("Prepare phase failed. Retrying...")
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if highestAcceptedValue == nil {
			highestAcceptedValue = value
		}

		if p.acceptPhase(proposalID, highestAcceptedValue) {
			fmt.Println("Proposal accepted:", highestAcceptedValue)
			break
		}

		fmt.Println("Accept phase failed. Retrying...")
		time.Sleep(backoff)
		backoff *= 2
	}
}

func (p *Proposer) preparePhase(proposalID ProposalID) (interface{}, bool) {
	prepareRequest := PrepareRequest{ProposalID: proposalID}
	responses := make(chan PrepareResponse, len(p.acceptors))
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	remaining := len(p.acceptors)
	successCount := 0
	var highestAcceptedID ProposalID
	var highestAcceptedValue interface{}

	for _, acceptor := range p.acceptors {
		wg.Add(1)
		go func(a *Acceptor) {
			defer wg.Done()
			response := a.HandlePrepare(prepareRequest)
			select {
			case responses <- response:
			case <-ctx.Done():
			}
		}(acceptor)
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	for remaining > 0 {
		select {
		case resp, ok := <-responses:
			if !ok {
				return highestAcceptedValue, successCount >= p.quorum
			}
			remaining--
			if resp.Ok {
				successCount++
				if resp.AcceptedID.GreaterThanOrEqual(highestAcceptedID) {
					highestAcceptedID = resp.AcceptedID
					highestAcceptedValue = resp.AcceptedValue
				}
			}

			if successCount >= p.quorum {
				return highestAcceptedValue, true
			}

			if remaining+successCount < p.quorum {
				return highestAcceptedValue, false
			}

		case <-ctx.Done():
			return highestAcceptedValue, successCount >= p.quorum
		}
	}

	return highestAcceptedValue, successCount >= p.quorum
}

func (p *Proposer) acceptPhase(proposalID ProposalID, value interface{}) bool {
	acceptRequest := AcceptRequest{ProposalID: proposalID, Value: value}
	var mu sync.Mutex
	successCount := 0
	var wg sync.WaitGroup

	for _, acceptor := range p.acceptors {
		wg.Add(1)
		go func(a *Acceptor) {
			defer wg.Done()
			response := a.HandleAccept(acceptRequest)
			if response.Ok {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(acceptor)
	}

	wg.Wait()
	return successCount >= p.quorum
}
