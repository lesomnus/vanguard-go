package control

import (
	"encoding/json"
	"fmt"

	"github.com/lesomnus/vanguard-go/signal"
)

type Envelope struct {
	Body Message
}

type Message interface {
	kind() string
}

func (m *Envelope) MarshalJSON() ([]byte, error) {
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
	case "closing":
		m.Body = &Closing{}
	case "close-ack":
		m.Body = &CloseAck{}
	case "connect":
		m.Body = &Connect{}
	case "accept":
		m.Body = &Accept{}
	case "reject":
		m.Body = &Reject{}
	case "signal":
		m.Body = &Signal{}
	default:
		return fmt.Errorf("unknown type of signalling message: %s", k.Kind)
	}

	if err := json.Unmarshal(b, m.Body); err != nil {
		return err
	}

	return nil
}

type Closing struct {
}

func (s *Closing) kind() string {
	return "closing"
}

type CloseAck struct {
}

func (s *CloseAck) kind() string {
	return "close-ack"
}

type Connect struct {
	Port  uint32 `json:"port"`
	Label string `json:"label"`
}

func (s *Connect) kind() string {
	return "connect"
}

type Accept struct {
	Port uint32 `json:"port"`
}

func (s *Accept) kind() string {
	return "accept"
}

type Reject struct {
	Port uint32 `json:"port"`
}

func (s *Reject) kind() string {
	return "reject"
}

type Signal struct {
	Port    uint32          `json:"port"`
	Message signal.Envelope `json:"message"`
}

func (s *Signal) kind() string {
	return "signal"
}
