package transport

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
)

const defaultMaxEnvelopeSize = 4 * 1024 * 1024

var maxEnvelopeSize uint32 = defaultMaxEnvelopeSize

func SetMaxEnvelopeSize(size uint32) {
	if size == 0 {
		maxEnvelopeSize = defaultMaxEnvelopeSize
		return
	}
	maxEnvelopeSize = size
}

func ReadEnvelope(conn net.Conn) (*internalpb.Envelope, error) {
	return readEnvelope(conn)
}

func WriteEnvelope(conn net.Conn, env *internalpb.Envelope) error {
	return writeEnvelope(conn, env)
}

func readEnvelope(reader io.Reader) (*internalpb.Envelope, error) {
	var sizeBuf [4]byte
	if _, err := io.ReadFull(reader, sizeBuf[:]); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(sizeBuf[:])
	if maxEnvelopeSize > 0 && size > maxEnvelopeSize {
		return nil, fmt.Errorf("envelope too large: %d > %d", size, maxEnvelopeSize)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	var env internalpb.Envelope
	if err := proto.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func writeEnvelope(writer io.Writer, env *internalpb.Envelope) error {
	data, err := proto.Marshal(env)
	if err != nil {
		return err
	}

	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], uint32(len(data)))

	if _, err := writer.Write(sizeBuf[:]); err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}
