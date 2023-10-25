package serialize

type Serializable interface {
}

func Serialize(obj Serializable) []byte {
	return []byte{} //TODO
}

func Deserialize(data []byte) Serializable {
	return nil //TODO
}