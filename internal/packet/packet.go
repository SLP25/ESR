package packet

import (
	"bytes"
	"encoding/json"
	"log/slog"
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
	reflect.TypeOf(StreamCancel{}),
	reflect.TypeOf(StreamEnd{}),
	reflect.TypeOf(StreamPacket{}),
}

func fetchType(name string) reflect.Type {
	for _, t := range packet_list {
		if t.Name() == name {
			return t
		}
	}

	return nil
}

//TODO: space-efficient marshalling

func Serialize(val Packet) []byte {
	b, err := json.Marshal(val)
    if err != nil {
        slog.Error("Error serializing packet", "err", err)
        return []byte{}
    }
	return append(append([]byte(reflect.TypeOf(val).Name()), b...), 0)
}

func Deserialize(data []byte) Packet {
	//Remove trailing NULL byte
	data = data[:len(data) - 1]
	aux := bytes.SplitN(data, []byte("{"), 2)
	pType := fetchType(string(aux[0]))
	val := reflect.New(pType).Interface()

	err := json.Unmarshal(append([]byte("{"), aux[1]...), val)
    if err != nil {
		slog.Error("Error deserializing packet", "err", err)
        return nil
    }

	return reflect.ValueOf(val).Elem().Interface()
}