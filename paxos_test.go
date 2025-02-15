package paxos

import (
	"context"
	"testing"
	"time"
)

func TestPaxosWithSingleProposer(t *testing.T) {
	// 1, 2, 3 are acceptors
	// 1001 is acceptor proposer
	ctx := context.Background()
	pn := NewNetwork(1, 2, 3, 1001, 2001)

	acceptors := make([]IAcceptor, 0)
	for i := 1; i <= 3; i++ {
		acceptors = append(acceptors, NewAcceptor(uint32(i), pn, []uint32{2001}))
	}

	for _, acceptor := range acceptors {
		go func() {
			err := acceptor.Run(ctx)
			if err != nil {
				t.Errorf("Error while running acceptor: %#v, error: %#v", acceptor, err)
			}
		}()
	}

	proposer := NewProposer(1001, pn, []uint32{1, 2, 3})
	go func() {
		_, err := proposer.Run(ctx, "hello world")
		if err != nil {
			t.Errorf("Error while running proposer: %#v, error: %#v", proposer, err)
		}
	}()

	l := NewLearner(2001, pn, []uint32{1, 2, 3})
	value, ok := l.Run(ctx)
	if !ok {
		t.Errorf("value = %s, want %s", value, "hello world")
	}
}

func TestPaxosWithTwoProposers(t *testing.T) {
	// 1, 2, 3 are acceptors
	// 1001,1002 is acceptor proposer
	ctx := context.Background()
	pn := NewNetwork(1, 2, 3, 1001, 1002, 2001)

	acceptors := make([]IAcceptor, 0)
	for i := 1; i <= 3; i++ {
		acceptors = append(acceptors, NewAcceptor(uint32(i), pn, []uint32{2001}))
	}

	for _, acceptor := range acceptors {
		go func() {
			err := acceptor.Run(ctx)
			if err != nil {
				t.Errorf("Error while running acceptor: %#v, error: %#v", acceptor, err)
			}
		}()
	}

	proposer1 := NewProposer(1001, pn, []uint32{1, 2, 3})
	go func() {
		_, err := proposer1.Run(ctx, "hello world")
		if err != nil {
			t.Errorf("Error while running proposer1: %#v, error: %#v", proposer1, err)
		}
	}()

	time.Sleep(time.Millisecond)
	proposer2 := NewProposer(1002, pn, []uint32{1, 2, 3})
	go func() {
		_, err := proposer2.Run(ctx, "bad day")
		if err != nil {
			t.Errorf("Error while running proposer2: %#v, error: %#v", proposer2, err)
		}
	}()

	l := NewLearner(2001, pn, []uint32{1, 2, 3})
	value, ok := l.Run(ctx)
	if !ok {
		t.Errorf("value = %s, want %s", value, "hello world")
	}
	time.Sleep(time.Millisecond)
}
