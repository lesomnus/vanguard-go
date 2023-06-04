package control_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/lesomnus/vanguard-go/control"
	"github.com/lesomnus/vanguard-go/signal"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	t.Run("sanity", func(t *testing.T) {
		testCases := []struct {
			body control.Message
		}{
			{body: &control.Closing{}},
			{body: &control.CloseAck{}},
			{body: &control.Connect{}},
			{body: &control.Accept{}},
			{body: &control.Reject{}},
			{body: &control.Signal{Message: signal.Envelope{Body: &signal.Abort{}}}},
		}
		for _, tC := range testCases {
			t.Run(reflect.TypeOf(tC.body).Elem().Name(), func(t *testing.T) {
				require := require.New(t)

				msg := control.Envelope{Body: tC.body}
				data, err := json.Marshal(&msg)
				require.NoError(err)

				msg = control.Envelope{}
				err = json.Unmarshal(data, &msg)
				require.NoError(err)
				require.IsType(tC.body, msg.Body)
			})
		}
	})

	t.Run("unknown kind", func(t *testing.T) {
		require := require.New(t)

		msg := control.Envelope{}
		err := json.Unmarshal([]byte(`{"kind":"foo"}`), &msg)
		require.ErrorContains(err, "unknown")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		require := require.New(t)

		msg := control.Envelope{}
		err := json.Unmarshal([]byte(`{kind: foo}`), &msg)
		expected := &json.SyntaxError{}
		require.ErrorAs(err, &expected)
	})

	t.Run("invalid body", func(t *testing.T) {
		require := require.New(t)

		msg := control.Envelope{}
		err := json.Unmarshal([]byte(`{"kind":"connect","port":"foo"}`), &msg)
		expected := &json.UnmarshalTypeError{}
		require.ErrorAs(err, &expected)
	})

	t.Run("nested with signal message", func(t *testing.T) {
		require := require.New(t)

		msg := control.Envelope{Body: &control.Signal{
			Port:    42,
			Message: signal.Envelope{Body: &signal.Abort{Reason: "foo"}},
		}}
		data, err := json.Marshal(&msg)
		require.NoError(err)

		msg_parsed := control.Envelope{}
		err = json.Unmarshal(data, &msg_parsed)
		require.NoError(err)
		require.Equal(msg, msg_parsed)
	})
}
