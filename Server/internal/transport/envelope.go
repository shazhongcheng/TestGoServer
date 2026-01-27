package transport

import (
	"encoding/binary"
	"io"
	"net"

	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
)

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
