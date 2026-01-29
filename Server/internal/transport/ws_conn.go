package transport

import (
	"game-server/internal/protocol/internalpb"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type WSConn struct {
	conn    *websocket.Conn
	useJSON bool
}

func NewWSConn(conn *websocket.Conn, useJSON bool) *WSConn {
	return &WSConn{
		conn:    conn,
		useJSON: useJSON,
	}
}

func (c *WSConn) ReadEnvelope() (*internalpb.Envelope, error) {
	messageType, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	var env internalpb.Envelope
	switch messageType {
	case websocket.TextMessage:
		if err := protojson.Unmarshal(data, &env); err != nil {
			return nil, err
		}
	default:
		if err := proto.Unmarshal(data, &env); err != nil {
			return nil, err
		}
	}
	return &env, nil
}

func (c *WSConn) WriteEnvelope(env *internalpb.Envelope) error {
	if c.useJSON {
		data, err := protojson.Marshal(env)
		if err != nil {
			return err
		}
		return c.conn.WriteMessage(websocket.TextMessage, data)
	}
	data, err := proto.Marshal(env)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *WSConn) Close() error {
	return c.conn.Close()
}
