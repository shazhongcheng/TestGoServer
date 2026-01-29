package transport

import (
	"bufio"
	"net"
	"sync"
	"time"

	"game-server/internal/protocol/internalpb"
)

const defaultBufferSize = 4 * 1024 * 1024

type BufferedConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex

	readTimeout  time.Duration
	writeTimeout time.Duration
}

type ConnOptions struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeepAlive    time.Duration
}

func NewBufferedConn(conn net.Conn) *BufferedConn {
	return NewBufferedConnWithOptions(conn, ConnOptions{})
}

func NewBufferedConnWithOptions(conn net.Conn, opts ConnOptions) *BufferedConn {
	if tcpConn, ok := conn.(*net.TCPConn); ok && opts.KeepAlive > 0 {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(opts.KeepAlive)
	}
	return &BufferedConn{
		conn:         conn,
		reader:       bufio.NewReaderSize(conn, defaultBufferSize),
		writer:       bufio.NewWriterSize(conn, defaultBufferSize),
		readTimeout:  opts.ReadTimeout,
		writeTimeout: opts.WriteTimeout,
	}
}

func (c *BufferedConn) ReadEnvelope() (*internalpb.Envelope, error) {
	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	return readEnvelope(c.reader)
}

func (c *BufferedConn) WriteEnvelope(env *internalpb.Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.writeTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	if err := writeEnvelope(c.writer, env); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *BufferedConn) Close() error {
	return c.conn.Close()
}
