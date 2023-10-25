package packet

import serialize "github.com/SLP25/ESR/internal/serialize"

type Packet interface {
	serialize.Serializable
}