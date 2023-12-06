package packet

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"log/slog"
	"math"
	"reflect"
)

type Packet interface {
}

var packet_list = []reflect.Type{
	reflect.TypeOf(StartupRequest{}),
	reflect.TypeOf(StartupResponseClient{}),
	reflect.TypeOf(StartupResponseNode{}),

	reflect.TypeOf(ProbeRequest{}),
	reflect.TypeOf(ProbeResponse{}),
	
	reflect.TypeOf(StreamRequest{}),
	reflect.TypeOf(StreamResponse{}),
	reflect.TypeOf(StreamCancel{}),
	reflect.TypeOf(StreamEnd{}),
	reflect.TypeOf(StreamPacket{}),
}

func encodeType(t reflect.Type) (byte, error) {
	for i, t2 := range packet_list {
		if t == t2 {
			return byte(i), nil
		}
	}

	slog.Error("Packet type not registered", "type", t)
	return 0, errors.New("Packet type " + t.Name() + " not registered")
}

func fetchType(name string) reflect.Type {
	for _, t := range packet_list {
		if t.Name() == name {
			return t
		}
	}

	slog.Error("Unknown packet type", "type", name)
	return nil
}

func packetToBytes(val Packet) ([]byte, error) {
	var byteBuffer bytes.Buffer 
	enc := gob.NewEncoder(&byteBuffer)
	err := enc.Encode(val)

	if err != nil {
		return make([]byte, 1), err
	} else {
		return byteBuffer.Bytes(), nil
	}
}

func bytesToPacket(bt []byte, val any) error {
	buf := bytes.NewBuffer(bt)
	dec := gob.NewDecoder(buf)

	return dec.Decode(&val)
}

func Serialize(val Packet, w io.Writer) (int, error) {
	b, err := packetToBytes(val)
    if err != nil {
        slog.Error("Error serializing packet", "err", err)
        return 0, err
    }

	if len(b) > math.MaxUint16 {
		return 0, errors.New("packet.Serialize(): Packet too big")
	}

	length := make([]byte, 2)
	binary.LittleEndian.PutUint16(length, uint16(len(b)))

	typeCode, err := encodeType(reflect.TypeOf(val))
	if err != nil { return 0, err }
	
	ans := append(append(length, typeCode), b...)
	_, err = w.Write(ans)
	if err != nil { return 0, err }

	return 3 + len(b), nil
}

func Deserialize(r io.Reader) (Packet, error) {
	var length uint16
	err := binary.Read(r, binary.LittleEndian, &length)
	if err != nil { return nil, err }

	var typeCode byte
	err = binary.Read(r, binary.LittleEndian, &typeCode)
	if err != nil { return nil, err }

	data := make([]byte, length)
	_, err = io.ReadFull(r, data)
	if err != nil { return nil, err }

	pType := packet_list[typeCode]
	val := reflect.New(pType).Interface()

	err = bytesToPacket(data, val)
    if err != nil { return nil, err }

	return reflect.ValueOf(val).Elem().Interface(), nil
}