package paxos

import "fmt"

type PrepareRequest struct {
	Number int64
}

type PrepareResponse struct {
	Number         int64
	AcceptedNumber int64
	Ok             bool
	AcceptedValue  interface{}
}

type AcceptRequest struct {
	Number int64
	Value  interface{}
}

type AcceptResponse struct {
	Number int64
	Ok     bool
	Value  interface{}
}

type LearnResponse struct {
	Number int64
	Value  interface{}
}

type MessageType int

const (
	DumbMessage MessageType = iota
	PrepareMessage
	PromiseMessage
	AcceptMessage
	AcceptedMessage
	LearnMessage
)

type Message struct {
	From  uint32
	To    uint32
	Type  MessageType
	Value interface{}
}

func (m Message) String() string {
	return fmt.Sprintf("Message{from: %d, to: %d, msgType: %d, value: %#v}", m.From, m.To, m.Type, m.Value)
}
