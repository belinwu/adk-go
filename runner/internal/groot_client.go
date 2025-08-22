package internal

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
}

type Port struct {
	Name     string `json:"name,omitempty"`
	StreamID string `json:"streamId,omitempty"`
}

type ActionGraph struct {
	Actions []*Action `json:"actions,omitempty"`
	Outputs []*Port   `json:"outputs,omitempty"`
}

type Action struct {
	Name    string  `json:"name,omitempty"`
	Inputs  []*Port `json:"inputs,omitempty"`
	Outputs []*Port `json:"outputs,omitempty"`
	// TODO: Add configs.
}

type Chunk struct {
	MIMEType string `json:"mimeType,omitempty"`
	Data     []byte `json:"data,omitempty"`
	// TODO: Add metadata.
}

type StreamFrame struct {
	StreamID  string `json:"streamId,omitempty"`
	Data      *Chunk `json:"data,omitempty"`
	Continued bool   `json:"continued,omitempty"`
}

type executeActionsMsg struct {
	SessionID    string         `json:"sessionId,omitempty"`
	ActionGraph  *ActionGraph   `json:"actionGraph,omitempty"`
	StreamFrames []*StreamFrame `json:"streamFrames,omitempty"`
}

func NewClient(endpoint string, apiKey string) (*Client, error) {
	c, _, err := websocket.DefaultDialer.Dial(endpoint+"?key="+apiKey, nil)
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

type Session struct {
	c         *Client
	sessionID string
}

func (c *Client) OpenSession(sessionID string) (*Session, error) {
	// TODO(jbd) Start session for real.
	return &Session{c: c, sessionID: sessionID}, nil
}

func (s *Session) ExecuteActions(actions []*Action, outputs []string) error {
	frames := ([]*StreamFrame{
		{StreamID: "test", Data: &Chunk{MIMEType: "text/plain", Data: []byte("hello world")}},
	})

	if err := s.c.conn.WriteJSON(&executeActionsMsg{
		SessionID: s.sessionID,
		ActionGraph: &ActionGraph{
			Actions: []*Action{
				{
					Name:    "save_stream",
					Inputs:  []*Port{{Name: "input", StreamID: "test"}},
					Outputs: []*Port{{Name: "output", StreamID: "save_output"}},
				},
				// {
				// 	Name:    "restore_stream",
				// 	Outputs: []*Port{{Name: "output", StreamID: "test"}},
				// },
			},
			Outputs: []*Port{{Name: "output", StreamID: "test"}},
		},
		StreamFrames: frames,
	}); err != nil {
		return err
	}

	for {
		var resp executeActionsMsg
		_, message, err := s.c.conn.ReadMessage()
		if err != nil {
			return err
		}
		log.Printf("received: %s\n", message)
		if err := json.Unmarshal(message, &resp); err != nil {
			return err
		}
		log.Println(resp.StreamFrames)
	}
}

// func (s *Session) ExecuteADK(name string, inputs)
