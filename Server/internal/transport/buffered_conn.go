package transport

import (
	"bufio"
	"net"
	"sync"

	"game-server/internal/protocol/internalpb"
)

const defaultBufferSize = 32 * 1024

type BufferedConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex
}

func NewBufferedConn(conn net.Conn) *BufferedConn {
	return &BufferedConn{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, defaultBufferSize),
		writer: bufio.NewWriterSize(conn, defaultBufferSize),
	}
}

func (c *BufferedConn) ReadEnvelope() (*internalpb.Envelope, error) {
	return readEnvelope(c.reader)
}

func (c *BufferedConn) WriteEnvelope(env *internalpb.Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := writeEnvelope(c.writer, env); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *BufferedConn) Close() error {
	return c.conn.Close()
}
