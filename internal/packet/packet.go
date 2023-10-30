package packet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type Packet interface {
}

var packet_list = []reflect.Type{
	reflect.TypeOf(StartupRequest{}),
	reflect.TypeOf(StartupResponse{}),
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
        fmt.Println(err)
        return []byte{}
    }
	fmt.Println(val)
	return append(append([]byte(reflect.TypeOf(val).Name()), b...), 0)
}

func Deserialize(data []byte) Packet {
	//Remove trailing NULL byte
	data = data[:len(data) - 1]
	aux := bytes.SplitN(data, []byte("{"), 2)
	val := reflect.New(fetchType(string(aux[0]))).Interface()

	err := json.Unmarshal(append([]byte("{"), aux[1]...), val)
    if err != nil {
        fmt.Println(err)
        return nil
    }
	fmt.Println(val)
	return val
}