package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tidwall/resp"
)

const defaultListenAddr = ":5001"

type Config struct {
	ListenAddr string
}

type Message struct {
	data resp.Value
	peer *Peer
}

type Server struct {
	Config
	peers     map[*Peer]bool
	ln        net.Listener
	addPeerCh chan *Peer
	quitCh    chan struct{}
	msgCh     chan Message

	kv *KV
}

func NewServer(cfg Config) *Server {
	if len(cfg.ListenAddr) == 0 {
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		Config:    cfg,
		peers:     make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh:    make(chan struct{}),
		msgCh:     make(chan Message),
		kv:        NewKV(),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.ln = ln

	go s.loop()

	slog.Info("server running", "listernAddr", s.ListenAddr)

	return s.acceptLoop()
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.addPeerCh:
			s.peers[peer] = true

		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				slog.Error("raw message error", "err:", err)
			}
		case <-s.quitCh:
			fmt.Println("")
			slog.Info("The Server is being shut down gracefuly")
			return
		}
	}
}

func (s *Server) handleMessage(msg Message) error {
	cmd, err := parseCommand(msg.data)
	if err != nil {
		return err
	}
	switch v := cmd.(type) {
	case SetCommand:
		slog.Info("somebody wants to set a key into the hashtable", "key", v.key, "val", v.val)
		return s.kv.Set(v.key, v.val)
	
	case GetCommand:
		slog.Info("sombody wants to get a key from teh hashtable", "key", v.key)
		
		val , ok := s.kv.Get(v.key)
		if !ok{
			slog.Error("could not find key", "key", v.key)
		}
		_, err := msg.peer.Send(val)
		if err != nil {
			slog.Error("failed to send response", "err", err)
		}
	}
	return nil
}

func (s *Server) acceptLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			slog.Error("accept errror", "err", err)
			continue
		}
		go s.handleCon(conn)
	}
}

func (s *Server) handleCon(conn net.Conn) {
	peer := NewPeer(conn, s.msgCh)
	s.addPeerCh <- peer
	if err := peer.readLoop(); err != nil {
		slog.Error("peer read error", "err", err, "remoteAddr", conn.RemoteAddr())
	}

}

func main() {
	server := NewServer(Config{})

	go func() {
		log.Fatal(server.Start())
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	time.Sleep(time.Second)
	
	<-sigCh
	server.quitCh <- struct{}{}
	time.Sleep(time.Second)
}
