package signal_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/lesomnus/vanguard-go/signal"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	t.Run("sanity", func(t *testing.T) {
		testCases := []struct {
			body signal.Message
		}{
			{body: &signal.Abort{}},
			{body: &signal.Sdp{}},
			{body: &signal.Candidate{}},
		}
		for _, tC := range testCases {
			t.Run(reflect.TypeOf(tC.body).Elem().Name(), func(t *testing.T) {
				require := require.New(t)

				msg := signal.Envelope{Body: tC.body}
				data, err := json.Marshal(&msg)
				require.NoError(err)

				msg = signal.Envelope{}
				err = json.Unmarshal(data, &msg)
				require.NoError(err)
				require.IsType(tC.body, msg.Body)
			})
		}
	})

	t.Run("unknown kind", func(t *testing.T) {
		require := require.New(t)

		msg := signal.Envelope{}
		err := json.Unmarshal([]byte(`{"kind":"foo"}`), &msg)
		require.ErrorContains(err, "unknown")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		require := require.New(t)

		msg := signal.Envelope{}
		err := json.Unmarshal([]byte(`{kind: foo}`), &msg)
		expected := &json.SyntaxError{}
		require.ErrorAs(err, &expected)
	})

	t.Run("invalid body", func(t *testing.T) {
		require := require.New(t)

		msg := signal.Envelope{}
		err := json.Unmarshal([]byte(`{"kind":"abort","reason":42}`), &msg)
		expected := &json.UnmarshalTypeError{}
		require.ErrorAs(err, &expected)
	})
}
