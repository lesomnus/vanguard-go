# Vanguard

Make extra connections from the existing connection.

## Usage

```go
import (
	"github.com/lesomnus/vanguard-go"
	"github.com/lesomnus/vanguard-go/signal"
	"github.com/pion/webrtc/v3"
)

// Implements `signal.Channel`.
struct MySigChannel {
	// ...
}

func (c *MySigChannel) Send(message signal.Message) error {
	// Implement it.
}

func (c *MySigChannel) Recv() (signal.Message, error) {
	// Implement it.
}

func (c *MySigChannel) Close() error {
	// Implement it.
}

func main() {
	// Initial connection with third-party signalling server.
	conn, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	peer, err := vanguard.Offer(conn, &MySigChannel{})
	if err != nil {
		panic(err)
	}

	// Need extra connection?
	// Signaling through existing connection!
	extra_conn, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	extra_peer, err := peer.Offer(conn, "with_label")
	if err != nil {
		panic(err)
	}
}


```
