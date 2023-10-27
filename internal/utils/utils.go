package utils

type ServiceType byte

const (
	Bootstrapper ServiceType = iota
	Client
	Node
	Server
)
