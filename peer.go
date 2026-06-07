package main

import (
	"net"

	"github.com/tidwall/resp"
)

type Peer struct {
	conn  net.Conn
	msgCh chan resp.Value
}

func NewPeer(conn net.Conn, msgCh chan resp.Value) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
	}
}

func (p *Peer) readLoop() error {
	rd := resp.NewReader(p.conn)
	for {
		v, _, err := rd.ReadValue()
		if err != nil {
			return err
		}
		p.msgCh <- v
	}
}
