package signal

import (
	"context"
	"io"
)

type Channel interface {
	Send(message Message) error
	Recv() (Message, error)
	Close() error
}

type pipe struct {
	tx chan Message
	rx chan Message

	ctx    context.Context
	cancel context.CancelFunc
}

func newPipe(tx chan Message, rx chan Message) *pipe {
	ctx, cancel := context.WithCancel(context.Background())
	return &pipe{
		tx: tx,
		rx: rx,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *pipe) Send(message Message) error {
	select {
	case <-p.ctx.Done():
		return io.EOF
	case p.tx <- message:
		return nil
	}
}

func (p *pipe) Recv() (Message, error) {
	select {
	case <-p.ctx.Done():
		return nil, io.EOF
	case message := <-p.rx:
		return message, nil
	}
}

func (p *pipe) Close() error {
	p.cancel()
	return nil
}

func Pipe() (Channel, Channel) {
	c1 := make(chan Message)
	c2 := make(chan Message)

	return newPipe(c1, c2), newPipe(c2, c1)
}
