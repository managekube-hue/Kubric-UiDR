// Package messaging provides a ZMQ4 pub/sub bridge for high-throughput
// inter-service event fanout (e.g. NetGuard flow events → analytics workers).
// Uses the pure-Go zmq4 implementation — no libzmq CGO dependency.
package messaging

import (
	"context"
	"fmt"

	"github.com/go-zeromq/zmq4"
)

// Publisher wraps a ZMQ4 PUB socket.
type Publisher struct {
	sock zmq4.Socket
}

// NewPublisher binds a ZMQ4 PUB socket to addr (e.g. "tcp://0.0.0.0:5555").
func NewPublisher(ctx context.Context, addr string) (*Publisher, error) {
	sock := zmq4.NewPub(ctx)
	if err := sock.Listen(addr); err != nil {
		return nil, fmt.Errorf("zmq publisher listen %s: %w", addr, err)
	}
	return &Publisher{sock: sock}, nil
}

// Publish sends topic + payload on the PUB socket.
func (p *Publisher) Publish(topic string, payload []byte) error {
	msg := zmq4.NewMsgFrom([]byte(topic), payload)
	if err := p.sock.Send(msg); err != nil {
		return fmt.Errorf("zmq publish: %w", err)
	}
	return nil
}

// Close releases the socket.
func (p *Publisher) Close() error { return p.sock.Close() }

// Subscriber wraps a ZMQ4 SUB socket.
type Subscriber struct {
	sock zmq4.Socket
}

// NewSubscriber dials addr and subscribes to the given topic prefix.
func NewSubscriber(ctx context.Context, addr, topic string) (*Subscriber, error) {
	sock := zmq4.NewSub(ctx)
	if err := sock.Dial(addr); err != nil {
		return nil, fmt.Errorf("zmq subscriber dial %s: %w", addr, err)
	}
	if err := sock.SetOption(zmq4.OptionSubscribe, topic); err != nil {
		return nil, fmt.Errorf("zmq subscriber set topic: %w", err)
	}
	return &Subscriber{sock: sock}, nil
}

// Recv blocks until a message arrives and returns (topic, payload, error).
func (s *Subscriber) Recv() (string, []byte, error) {
	msg, err := s.sock.Recv()
	if err != nil {
		return "", nil, fmt.Errorf("zmq recv: %w", err)
	}
	if len(msg.Frames) < 2 {
		return "", nil, fmt.Errorf("zmq recv: expected 2 frames, got %d", len(msg.Frames))
	}
	return string(msg.Frames[0]), msg.Frames[1], nil
}

// Close releases the socket.
func (s *Subscriber) Close() error { return s.sock.Close() }
