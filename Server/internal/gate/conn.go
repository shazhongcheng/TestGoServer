// internal/gate/conn.go
package gate

import (
	"encoding/binary"
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"time"
)

type Conn struct {
	id        int64
	sessionID int64
	gate      *Gate
	netConn   net.Conn
}

func NewConn(nc net.Conn, g *Gate) *Conn {
	return &Conn{
		netConn: nc,
		gate:    g,
	}
}

func (c *Conn) readPacket() (*internalpb.Envelope, error) {
	var sizeBuf [4]byte
	if _, err := io.ReadFull(c.netConn, sizeBuf[:]); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(sizeBuf[:])
	data := make([]byte, size)

	if _, err := io.ReadFull(c.netConn, data); err != nil {
		return nil, err
	}

	var env internalpb.Envelope
	if err := proto.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

func (c *Conn) writeEnvelope(env *internalpb.Envelope) error {
	data, err := proto.Marshal(env)
	if err != nil {
		return err
	}

	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], uint32(len(data)))

	if _, err := c.netConn.Write(sizeBuf[:]); err != nil {
		return err
	}
	_, err = c.netConn.Write(data)
	return err
}

func (c *Conn) close() {
	_ = c.netConn.Close()
}

func (c *Conn) ReadLoop() {
	for {
		env, err := c.readPacket()
		if err != nil {
			c.gate.onConnClose(c)
			return
		}

		// 重要：首次连接时 env.session_id 可能为 0
		c.gate.OnEnvelope(c, env)
	}
}

func (g *Gate) onConnClose(c *Conn) {
	s := g.sessions.Get(c.sessionID)
	if s == nil {
		return
	}

	s.Conn = nil
	s.State = SessionOffline
	s.LastSeen = time.Now()

	g.notifyPlayerOffline(s)
}

func (g *Gate) onResume(c *Conn, req *ResumeReq) error {
	s := g.sessions.Get(req.SessionId)
	if s == nil {
		return protocol.InternalErrInvalidSession
	}

	if !g.verifyToken(s, req.Token) {
		return protocol.InternalErrInvalidToken
	}

	// 重新绑定
	s.Conn = c
	s.State = SessionOnline
	s.LastSeen = time.Now()
	c.sessionID = s.ID

	return nil
}

func (g *Gate) attachPlayer(sessionID int64, playerID int64) {
	s := g.sessions.Get(sessionID)
	if s == nil {
		return
	}
	if s.PlayerID == playerID {
		return
	}
	s.PlayerID = playerID
}
