package paxos

type ProposalID struct {
	NodeID   int
	Sequence int
}

func (p ProposalID) GreaterThanOrEqual(other ProposalID) bool {
	return p.Sequence >= other.Sequence || (p.Sequence == other.Sequence && p.NodeID >= other.NodeID)
}

type Proposal struct {
	ID    ProposalID
	Value interface{}
}

type PrepareRequest struct {
	ProposalID ProposalID
}

type PrepareResponse struct {
	Ok            bool
	AcceptedID    ProposalID
	AcceptedValue interface{}
}

type AcceptRequest struct {
	ProposalID ProposalID
	Value      interface{}
}

type AcceptResponse struct {
	Ok bool
} 