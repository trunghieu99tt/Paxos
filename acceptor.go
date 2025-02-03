package paxos

import "sync"

type Acceptor struct {
	mu                      sync.Mutex
	minProposalID           ProposalID
	highestAcceptedProposal *Proposal
	learners                []*Learner
}

func NewAcceptor(learners []*Learner) *Acceptor {
	return &Acceptor{
		learners: learners,
	}
}

func (a *Acceptor) HandlePrepare(request PrepareRequest) PrepareResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !request.ProposalID.GreaterThanOrEqual(a.minProposalID) {
		return PrepareResponse{Ok: false}
	}

	a.minProposalID = request.ProposalID
	if a.highestAcceptedProposal == nil {
		return PrepareResponse{
			Ok:            true,
			AcceptedID:    ProposalID{},
			AcceptedValue: nil,
		}
	}

	return PrepareResponse{
		Ok:            true,
		AcceptedID:    a.highestAcceptedProposal.ID,
		AcceptedValue: a.highestAcceptedProposal.Value,
	}
}

func (a *Acceptor) HandleAccept(request AcceptRequest) AcceptResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !request.ProposalID.GreaterThanOrEqual(a.minProposalID) {
		return AcceptResponse{Ok: false}
	}

	a.highestAcceptedProposal = &Proposal{
		ID:    request.ProposalID,
		Value: request.Value,
	}
	a.minProposalID = request.ProposalID
	for _, learner := range a.learners {
		go func(l *Learner) {
			l.Learn(request.Value)
		}(learner)
	}
	return AcceptResponse{Ok: true}
} 