package paxos

import (
	"context"
	"time"
)

type Proposer struct {
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

func (p *Proposer) prepare(ctx context.Context, value interface{}) (proposalValue interface{}, ok bool, err error) {
	p.sequence += 1
	proposalID := p.proposalNumber()
	for _, acceptorID := range p.acceptorNodeIds {
		msg := Message{
			From:  p.id,
			To:    acceptorID,
			Type:  PrepareMessage,
			Value: PrepareRequest{Number: proposalID},
		}
		if err := p.network.Send(ctx, msg); err != nil {
			continue
		}
	}

	receivedResp := make(map[uint32]bool)
	for !p.isPromiseQuorumReached() && len(receivedResp) < len(p.acceptorNodeIds) {
		select {
		case <-time.After(PrepareTimeout):
			return
		default:
			msg, err := p.network.Receive(ctx, p.id, MessageTimeout)
			if err == nil {
				prepareResponse := msg.Value.(PrepareResponse)
				if msg.Type == PromiseMessage && prepareResponse.Number == proposalID {
					if prepareResponse.Ok {
						receivedResp[msg.From] = true
					}
					if _, ok := p.acceptorPromises[msg.From]; ok {
						p.acceptorPromises[msg.From] = &msg
					}
				}
			}
		}
	}

	highestAcceptedValue := p.highestAcceptedValue(value)
	return highestAcceptedValue, true, nil
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
	okAcceptedResponseCnt := 0
	acceptedResponseCnt := 0
	for okAcceptedResponseCnt < int(p.quorum) && acceptedResponseCnt < len(p.acceptorNodeIds) {
		select {
		case <-time.After(AcceptTimeout):
			return acceptedResponseCnt >= int(p.quorum)
		default:
			msg, err := p.network.Receive(ctx, p.id, MessageTimeout)
			if err != nil {
				continue
			}
			if msg.Type == AcceptedMessage {
				acceptedResponseCnt += 1
				if msg.Value.(AcceptResponse).Ok {
					okAcceptedResponseCnt++
				}
			}
		}
	}
	return acceptedResponseCnt >= int(p.quorum)
}
