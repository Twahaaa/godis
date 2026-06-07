package main

import (
	"net"

	"github.com/tidwall/resp"
)

type Peer struct {
	conn  net.Conn
	msgCh chan Message
}


func NewPeer(conn net.Conn, msgCh chan Message) *Peer {
	return &Peer{
		conn:  conn,
		msgCh: msgCh,
	}
}

func (p *Peer) Send(msg []byte) (int, error){
	return p.conn.Write(msg)
}

func (p *Peer) readLoop() error {
	rd := resp.NewReader(p.conn)
	for {
		v, _, err := rd.ReadValue()
		if err != nil {
			return err
		}
		p.msgCh <- Message{
			data: v,
			peer: p,
		}
	}
}
