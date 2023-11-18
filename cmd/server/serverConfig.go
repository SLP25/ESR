package main

import (
	"encoding/json"
	"os"
)

type config map[string] string

func MustReadConfig(filename string) config {
	bytes, err := os.ReadFile(filename)
	if err != nil { panic(err.Error()) }
	
	conf := make(config)
	err = json.Unmarshal(bytes, &conf)
	if err != nil {
		panic("Error parsing server config: " + err.Error())
	}

	return conf
}

//func (this *config) getStream(id string) {
//	file := (*this)[id]
//	open file?
//}