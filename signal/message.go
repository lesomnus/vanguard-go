package signal

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pion/webrtc/v3"
)

type Envelope struct {
	Body Message
}

type Message interface {
	kind() string
}

func (m *Envelope) MarshalJSON() ([]byte, error) {
	if m.Body == nil {
		return nil, errors.New("empty body")
	}

	body, err := json.Marshal(m.Body)
	if err != nil {
		return nil, err
	}

	str := ""
	if string(body) == "{}" {
		str = fmt.Sprintf(`{"kind":"%s"}`, m.Body.kind())
	} else {
		str = fmt.Sprintf(`{"kind":"%s",%s`, m.Body.kind(), body[1:])
	}

	return []byte(str), nil
}

func (m *Envelope) UnmarshalJSON(b []byte) error {
	k := struct {
		Kind string `json:"kind"`
	}{}

	if err := json.Unmarshal(b, &k); err != nil {
		return err
	}

	switch k.Kind {
	case "abort":
		m.Body = &Abort{}
	case "sdp":
		m.Body = &Sdp{}
	case "candidate":
		m.Body = &Candidate{}
	default:
		return fmt.Errorf("unknown type of signalling message: %s", k.Kind)
	}

	if err := json.Unmarshal(b, m.Body); err != nil {
		return err
	}

	return nil
}

type Abort struct {
	Reason string `json:"reason"`
}

func (m *Abort) kind() string {
	return "abort"
}

type Sdp struct {
	Data *webrtc.SessionDescription `json:"data"`
}

func (m *Sdp) kind() string {
	return "sdp"
}

type Candidate struct {
	Data *webrtc.ICECandidate `json:"data"`
}

func (m *Candidate) kind() string {
	return "candidate"
}
