package paxos

import (
	"context"
	"log"
	"sync"
)

type IAcceptor interface {
	HandlePrepare(ctx context.Context, msg Message) error
	HandleAccept(ctx context.Context, msg Message) error
	Run(ctx context.Context) error
}

type Acceptor struct {
	mu             sync.RWMutex
	id             uint32
	minNumber      int64
	acceptedNumber int64
	network        INetwork
	learnerIds     []uint32
	acceptedValue  interface{}
}

func NewAcceptor(id uint32, network INetwork, learnerIds []uint32) IAcceptor {
	return &Acceptor{
		id:         id,
		network:    network,
		learnerIds: learnerIds,
	}
}

func (a *Acceptor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := a.network.Receive(ctx, a.id, MessageTimeout)
			if err != nil {
				continue
			}
			switch msg.Type {
			case PrepareMessage:
				if err := a.HandlePrepare(ctx, msg); err != nil {
					log.Printf("Error handling prepare message: %v", err)
				}
			case AcceptMessage:
				if err := a.HandleAccept(ctx, msg); err != nil {
					log.Printf("Error handling accept message: %v", err)
				}
			default:
				panic("unhandled default case")
			}
		}
	}
}

func (a *Acceptor) HandlePrepare(ctx context.Context, msg Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if msg.Value == nil {
		return nil
	}

	prepareRequest := msg.Value.(PrepareRequest)
	response := PrepareResponse{
		Number: prepareRequest.Number,
		Ok:     prepareRequest.Number >= a.minNumber,
	}

	if response.Ok {
		a.minNumber = prepareRequest.Number
		if a.acceptedValue != nil {
			response.AcceptedNumber = a.acceptedNumber
			response.AcceptedValue = a.acceptedValue
		}
	}

	return a.network.Send(ctx, Message{
		From:  a.id,
		To:    msg.From,
		Type:  PromiseMessage,
		Value: response,
	})
}

func (a *Acceptor) HandleAccept(ctx context.Context, msg Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	acceptRequest := msg.Value.(AcceptRequest)
	response := AcceptResponse{
		Number: a.acceptedNumber,
		Ok:     acceptRequest.Number >= a.minNumber,
	}

	if response.Ok {
		a.acceptedNumber = acceptRequest.Number
		a.acceptedValue = acceptRequest.Value
		response.Number = a.acceptedNumber
		response.Value = a.acceptedValue

		// Notify learners in background
		a.notifyLearners(ctx, response)
	}

	return a.network.Send(ctx, Message{
		From:  a.id,
		To:    msg.From,
		Type:  AcceptedMessage,
		Value: response,
	})
}

func (a *Acceptor) notifyLearners(ctx context.Context, response AcceptResponse) {
	for _, learnerID := range a.learnerIds {
		go func(learnerID uint32) {
			if err := a.network.Send(ctx, Message{
				From:  a.id,
				To:    learnerID,
				Type:  LearnMessage,
				Value: response,
			}); err != nil {
				log.Printf("Error sending learn message to %d: %v", learnerID, err)
			}
		}(learnerID)
	}
}
