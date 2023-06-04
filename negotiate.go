package vanguard

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/lesomnus/vanguard-go/signal"
	"github.com/pion/webrtc/v3"
)

func Dial(conn *webrtc.PeerConnection, sig signal.Channel, local func() (webrtc.SessionDescription, error)) (*Peer, error) {
	var (
		z = uint16(0)
		t = true
	)
	ctrl, err := conn.CreateDataChannel("_vanguard", &webrtc.DataChannelInit{ID: &z, Negotiated: &t})
	if err != nil {
		return nil, fmt.Errorf("create data channel: %w", err)
	}

	var result atomic.Value

	defer func() {
		conn.OnICECandidate(func(candidate *webrtc.ICECandidate) {})
		conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {})
		sig.Close()
	}()

	abort := func(err error) (*Peer, error) {
		result.Store(err)
		sig.Send(&signal.Abort{Reason: err.Error()})
		sig.Close()
		return nil, err
	}

	conn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		sig.Send(&signal.Candidate{Data: candidate})
	})
	conn.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateClosed:
			abort(fmt.Errorf("peer closed"))
		case webrtc.PeerConnectionStateFailed:
			abort(fmt.Errorf("peer failed"))
		case webrtc.PeerConnectionStateConnected:
			sig.Close()
		}
	})

	if sdp, err := local(); err != nil {
		return abort(fmt.Errorf("get local description: %w", err))
	} else if err := conn.SetLocalDescription(sdp); err != nil {
		return abort(fmt.Errorf("set local description: %w", err))
	}

	if sdp := conn.LocalDescription(); sdp == nil {
		return abort(fmt.Errorf("unexpected nil value for local description"))
	} else if err := sig.Send(&signal.Sdp{Data: sdp}); err != nil {
		return abort(fmt.Errorf("send local session description: %w", err))
	}

	for {
		message, err := sig.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return abort(fmt.Errorf("receive signal: %w", err))
			}
			if err, ok := result.Load().(error); ok && err != nil {
				return nil, err
			}
			if conn.ConnectionState() != webrtc.PeerConnectionStateConnected {
				return nil, fmt.Errorf("unexpected close of signalling channel")
			}

			return newPeer(conn, ctrl), nil
		}

		switch m := message.(type) {
		case *signal.Abort:
			return nil, fmt.Errorf("remote abort: %s", m.Reason)

		case *signal.Sdp:
			if err := conn.SetRemoteDescription(*m.Data); err != nil {
				return abort(fmt.Errorf("set remote description: %w", err))
			}

		case *signal.Candidate:
			if err := conn.AddICECandidate(m.Data.ToJSON()); err != nil {
				return abort(fmt.Errorf("add ICE candidate: %w", err))
			}
		}
	}
}

func Offer(conn *webrtc.PeerConnection, sig signal.Channel) (*Peer, error) {
	return Dial(conn, sig, func() (webrtc.SessionDescription, error) {
		return conn.CreateOffer(nil)
	})
}

func Answer(conn *webrtc.PeerConnection, sig signal.Channel) (*Peer, error) {
	return Dial(conn, sig, func() (webrtc.SessionDescription, error) {
		message, err := sig.Recv()
		if err != nil {
			return webrtc.SessionDescription{}, fmt.Errorf("receive session description: %w", err)
		}

		for {
			switch m := message.(type) {
			case *signal.Sdp:
				if err := conn.SetRemoteDescription(*m.Data); err != nil {
					return webrtc.SessionDescription{}, fmt.Errorf("set remote description: %w", err)
				}

				answer, err := conn.CreateAnswer(nil)
				if err != nil {
					return answer, fmt.Errorf("create answer: %w", err)
				}

				return answer, nil
			case *signal.Abort:
				return webrtc.SessionDescription{}, fmt.Errorf("remote abort: %s", m.Reason)
			}
		}
	})
}
