package signal_test

import (
	"io"
	"sync"
	"testing"

	"github.com/lesomnus/vanguard-go/signal"
	"github.com/stretchr/testify/require"
)

func TestPipe(t *testing.T) {
	require := require.New(t)

	c1, c2 := signal.Pipe()
	msgs := []signal.Message{}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			msg, err := c2.Recv()
			if err != nil {
				require.ErrorIs(err, io.EOF)
				return
			}

			msgs = append(msgs, msg)
		}
	}()

	err := c1.Send(&signal.Abort{})
	require.NoError(err)

	err = c1.Close()
	require.NoError(err)
	require.Equal([]signal.Message{&signal.Abort{}}, msgs)

	err = c1.Send(&signal.Abort{})
	require.ErrorIs(err, io.EOF)
}
