package paxos

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Proposer struct {
	mu               sync.Mutex
	sequence         uint64
	id               uint32
	quorum           uint32
	network          INetwork
	acceptorNodeIds  []uint32
	acceptorPromises map[uint32]*Message
}

type IProposer interface {
	Run(ctx context.Context, value interface{}) (interface{}, error)
}

func NewProposer(id uint32, network INetwork, acceptors []uint32) IProposer {
	acceptorPromises := make(map[uint32]*Message, len(acceptors))
	for _, acceptorID := range acceptors {
		acceptorPromises[acceptorID] = &Message{Type: DumbMessage}
	}
	return &Proposer{
		quorum:           uint32(len(acceptors)/2 + 1),
		id:               id,
		sequence:         0,
		network:          network,
		acceptorNodeIds:  acceptors,
		acceptorPromises: acceptorPromises,
	}
}

func (p *Proposer) Run(ctx context.Context, value interface{}) (interface{}, error) {
	for {
		select {
		case <-time.After(ProposerRunDur):
			return nil, nil
		default:
			proposalValue, ok, err := p.prepare(ctx, value)
			if err != nil {
				return nil, err
			}

			// Prepare phase failed, backoff and retry
			if !ok {
				time.Sleep(ProposerBackoff)
				ProposerBackoff *= BackoffFactor
				continue
			}

			p.sendAccept(ctx, proposalValue)
			if p.isConsensusReached(ctx) {
				return proposalValue, nil
			}

			// Accept phase failed, backoff and retry
			time.Sleep(ProposerBackoff)
			ProposerBackoff *= BackoffFactor
		}
	}
}

func (p *Proposer) prepare(ctx context.Context, value interface{}) (interface{}, bool, error) {
	p.mu.Lock()
	p.sequence++
	proposalID := p.proposalNumber()
	p.mu.Unlock()

	// Send prepare messages to all acceptors
	if err := p.broadcastPrepare(ctx, proposalID); err != nil {
		return nil, false, fmt.Errorf("failed to broadcast prepare: %w", err)
	}

	// Collect responses with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, PrepareTimeout)
	defer cancel()

	responses := make(map[uint32]bool)
	for len(responses) < len(p.acceptorNodeIds) {
		msg, err := p.network.Receive(timeoutCtx, p.id, MessageTimeout)
		if err != nil {
			break // Timeout or error occurred
		}

		if !p.isValidPrepareResponse(msg, proposalID) {
			continue
		}

		prepareResponse := msg.Value.(PrepareResponse)
		p.mu.Lock()
		if prepareResponse.Ok {
			responses[msg.From] = true
			p.acceptorPromises[msg.From] = &msg
		}
		hasQuorum := len(responses) >= int(p.quorum)
		p.mu.Unlock()

		if hasQuorum {
			return p.highestAcceptedValue(value), true, nil
		}
	}

	return nil, false, nil
}

func (p *Proposer) broadcastPrepare(ctx context.Context, proposalID int64) error {
	for _, acceptorID := range p.acceptorNodeIds {
		msg := Message{
			From:  p.id,
			To:    acceptorID,
			Type:  PrepareMessage,
			Value: PrepareRequest{Number: proposalID},
		}
		if err := p.network.Send(ctx, msg); err != nil {
			log.Printf("Failed to send prepare to acceptor %d: %v", acceptorID, err)
		}
	}
	return nil
}

func (p *Proposer) isValidPrepareResponse(msg Message, proposalID int64) bool {
	return msg.Type == PromiseMessage &&
		msg.Value.(PrepareResponse).Number == proposalID
}

func (p *Proposer) highestAcceptedValue(initialValue interface{}) interface{} {
	highestAcceptedID := int64(0)
	highestAcceptedValue := initialValue
	for _, promise := range p.acceptorPromises {
		if promise.Value == nil {
			continue
		}
		response := promise.Value.(PrepareResponse)
		if response.AcceptedNumber > highestAcceptedID {
			highestAcceptedID = response.Number
			highestAcceptedValue = response.AcceptedValue
		}
	}
	return highestAcceptedValue
}

func (p *Proposer) sendAccept(ctx context.Context, value interface{}) {
	for _, acceptorID := range p.acceptorNodeIds {
		msg := Message{
			From:  p.id,
			To:    acceptorID,
			Type:  AcceptMessage,
			Value: AcceptRequest{Number: p.proposalNumber(), Value: value},
		}
		if err := p.network.Send(ctx, msg); err != nil {
			continue
		}
	}
}

func (p *Proposer) isPromiseQuorumReached() bool {
	acceptedCount := 0
	for _, promise := range p.acceptorPromises {
		if promise.Type == PromiseMessage && promise.Value.(PrepareResponse).Number == p.proposalNumber() {
			acceptedCount++
		}
	}
	return acceptedCount >= int(p.quorum)
}

func (p *Proposer) proposalNumber() int64 { return int64(p.sequence)<<16 | int64(p.id) }

func (p *Proposer) isConsensusReached(ctx context.Context) bool {
	type acceptStats struct {
		total int
		ok    int
	}

	stats := &acceptStats{}
	timeoutCtx, cancel := context.WithTimeout(ctx, AcceptTimeout)
	defer cancel()

	for stats.ok < int(p.quorum) && stats.total < len(p.acceptorNodeIds) {
		msg, err := p.network.Receive(timeoutCtx, p.id, MessageTimeout)
		if err != nil {
			break
		}

		if msg.Type != AcceptedMessage {
			continue
		}

		p.mu.Lock()
		stats.total++
		if msg.Value.(AcceptResponse).Ok {
			stats.ok++
		}
		p.mu.Unlock()
	}

	return stats.total >= int(p.quorum)
}
