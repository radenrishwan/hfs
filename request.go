package hfs

import (
	"context"
	"net"
)

type Request struct {
	Context context.Context
	Method  string
	Path    string
	Version string
	Body    string
	Args    map[string]string
	Headers map[string]string
	Cookie  map[string]string
	Conn    net.Conn
}
