# Go Paxos Implementation

A clean, efficient implementation of the Paxos consensus algorithm in Go. This implementation provides a foundation for building distributed systems that require consensus among multiple nodes.

## Overview

Paxos is a family of protocols for solving consensus in a network of unreliable processors. This implementation includes:

- **Proposer**: Proposes values and drives the consensus process
- **Acceptor**: Accepts or rejects proposals based on the Paxos protocol rules
- **Learner**: Learns and stores the agreed-upon value

## Components

### Proposer
```go
// Create a proposer
proposer := NewProposer(nodeID, acceptors)

// Propose a value
proposer.Propose("Hello, Paxos!")
```

### Acceptor
```go
// Create an acceptor
acceptor := NewAcceptor(learners)

// Handles prepare and accept requests automatically
```

### Learner
```go
// Create a learner
learner := NewLearner()

// Get the learned value
value := learner.GetValue()
```

## Example Usage

```go
// Setup network
numAcceptors := 3
learner := NewLearner()
learners := []*Learner{learner}

var acceptors []*Acceptor
for i := 0; i < numAcceptors; i++ {
    acceptor := NewAcceptor(learners)
    acceptors = append(acceptors, acceptor)
}

// Create proposer and propose value
proposer := NewProposer(1, acceptors)
proposer.Propose("Hello, Paxos!")

// Get consensus value
result := learner.GetValue()
```

## Features

- Full implementation of the Paxos consensus algorithm
- Thread-safe operations
- Support for multiple proposers
- Automatic retry with exponential backoff
- Comprehensive test suite

## Testing

Run the tests:
```bash
go test -v
```

The test suite includes:
- Basic consensus verification
- Multiple competing proposers
- Proposal ID ordering
- Edge cases and failure scenarios

## Implementation Details

### Phase 1 (Prepare)
1. Proposer generates unique proposal ID
2. Proposer sends prepare requests to acceptors
3. Acceptors respond with promise or rejection

### Phase 2 (Accept)
1. If majority promises received, proposer sends accept requests
2. Acceptors accept if no higher proposal ID seen
3. Accepted values propagated to learners

## Requirements

- Go 1.16+
- github.com/stretchr/testify (for testing)

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## References
