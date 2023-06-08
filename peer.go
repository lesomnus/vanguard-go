package vanguard

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lesomnus/vanguard-go/control"
	"github.com/lesomnus/vanguard-go/signal"
	"github.com/pion/webrtc/v3"
)

type Peer struct {
	*webrtc.PeerConnection

	ctrl_mtx sync.Mutex
	ctrl     *webrtc.DataChannel

	channels *sync.Map
	onOffer  atomic.Value // func(string, signal.Channel)
}

func newPeer(conn *webrtc.PeerConnection, ctrl *webrtc.DataChannel) *Peer {
	channels := &sync.Map{}
	peer := &Peer{
		PeerConnection: conn,
		ctrl:           ctrl,

		channels: channels,
		onOffer:  atomic.Value{},
	}

	peer.onOffer.Store(func(_ string, sig signal.Channel) {
		time.Sleep(3 * time.Second)
		sig.Close()
	})

	ctrl.OnMessage(func(message webrtc.DataChannelMessage) {
		ctrl_msg := control.Envelope{}
		if err := json.Unmarshal(message.Data, &ctrl_msg); err != nil {
			return
		}

		switch m := ctrl_msg.Body.(type) {
		case (*control.Closing):
			{
				peer.sendControl(&control.CloseAck{})
				conn.Close()
			}
		case (*control.CloseAck):
			{
				conn.Close()
			}
		case (*control.Connect):
			{
				ctx, cancel := context.WithCancel(context.Background())
				sig := &sigChannel{
					port: m.Port,
					peer: peer,

					ctx:    ctx,
					cancel: cancel,

					buff: make(chan signal.Message, 10),
				}

				_, is_loaded := channels.LoadOrStore(m.Port, sig)
				if is_loaded {
					peer.sendControl(&control.Reject{Port: m.Port})
					return
				}

				peer.sendControl(&control.Accept{Port: m.Port})

				handler := peer.onOffer.Load().(func(label string, sig signal.Channel))
				handler(m.Label, sig)
			}

		case (*control.Accept):
			{
				c, ok := channels.Load(m.Port)
				if !ok {
					return
				}

				c.(*sigChannel).buff <- nil
			}

		case (*control.Reject):
			{
				c, ok := channels.LoadAndDelete(m.Port)
				if !ok {
					return
				}

				close(c.(*sigChannel).buff)
			}

		case (*control.Signal):
			{
				c, ok := channels.Load(m.Port)
				if !ok {
					return
				}

				// WARN: It will block underlying loop if the buffer is full.
				// Buffer should be consumed as soon as possible.
				c.(*sigChannel).buff <- m.Message.Body
			}

		default:
			panic("unknown type of control message")
		}
	})
	ctrl.OnClose(func() {
		channels.Range(func(_, c any) bool {
			c.(*sigChannel).Close()
			return true
		})
	})

	return peer
}

func (p *Peer) OnOffer(handler func(label string, sig signal.Channel)) {
	p.onOffer.Swap(handler)
}

func (p *Peer) Offer(conn *webrtc.PeerConnection, label string) (*Peer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sig := &sigChannel{
		peer: p,

		ctx:    ctx,
		cancel: cancel,
	}

	for {
		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
		if err != nil {
			cancel()
			return nil, fmt.Errorf("get random number: %w", err)
		}

		sig.port = uint32(n.Uint64())
		sig.buff = make(chan signal.Message, 10)
		_, is_loaded := p.channels.LoadOrStore(sig.port, sig)
		if is_loaded {
			// Port already exist.
			continue
		}

		if err := p.sendControl(&control.Connect{Port: sig.port, Label: label}); err != nil {
			cancel()
			return nil, fmt.Errorf("send: %w", err)
		}

		select {
		case <-sig.ctx.Done():
			{
				return nil, fmt.Errorf("signaling channel is closed while connecting")
			}

		case _, ok := <-sig.buff:
			if !ok {
				// Rejected.
				continue
			}
		}

		if peer, err := Offer(conn, sig); err != nil {
			cancel()
			return nil, err
		} else {
			return peer, nil
		}
	}
}

func (p *Peer) sendControl(message control.Message) error {
	p.ctrl_mtx.Lock()
	defer p.ctrl_mtx.Unlock()

	data, err := json.Marshal(&control.Envelope{Body: message})
	if err != nil {
		return err
	}

	return p.ctrl.SendText(string(data))
}

type sigChannel struct {
	port uint32
	peer *Peer

	ctx    context.Context
	cancel context.CancelFunc
	buff   chan signal.Message
}

func (c *sigChannel) Send(message signal.Message) error {
	return c.peer.sendControl(&control.Signal{Port: c.port, Message: signal.Envelope{Body: message}})
}

func (c *sigChannel) Recv() (signal.Message, error) {
	select {
	case <-c.ctx.Done():
		return nil, io.EOF
	case message := <-c.buff:
		return message, nil
	}
}

func (c *sigChannel) Close() error {
	c.cancel()
	c.peer.channels.LoadAndDelete(c.port)
	return nil
}
