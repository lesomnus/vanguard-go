package vanguard_test

import (
	"sync"
	"testing"

	"github.com/lesomnus/vanguard-go"
	"github.com/lesomnus/vanguard-go/signal"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/require"
)

func TestNegotiate(t *testing.T) {
	require := require.New(t)

	pc1, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(err)

	pc2, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(err)

	sig1, sig2 := signal.Pipe()
	peers := []*vanguard.Peer{nil, nil, nil, nil}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		peer, err := vanguard.Offer(pc1, sig1)
		require.NoError(err)

		peers[0] = peer
	}()
	go func() {
		defer wg.Done()

		peer, err := vanguard.Answer(pc2, sig2)
		require.NoError(err)

		peers[1] = peer
	}()
	wg.Wait()

	require.Equal(webrtc.PeerConnectionStateConnected, peers[0].ConnectionState())
	require.Equal(webrtc.PeerConnectionStateConnected, peers[1].ConnectionState())

	// Make an extra connection.
	pc3, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(err)

	pc4, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(err)

	wg.Add(2)
	peers[1].OnOffer(func(sig signal.Channel) {
		go func() {
			defer wg.Done()
			peer, err := vanguard.Answer(pc4, sig)
			require.NoError(err)

			peers[3] = peer
		}()
	})
	go func() {
		defer wg.Done()

		peer, err := peers[0].Offer(pc3)
		require.NoError(err)

		peers[2] = peer
	}()
	wg.Wait()

	require.Equal(webrtc.PeerConnectionStateConnected, peers[2].ConnectionState())
	require.Equal(webrtc.PeerConnectionStateConnected, peers[3].ConnectionState())
}
