package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
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
		var ttl time.Duration
		if len(v.ttl) > 0 {
			ttl, err = parseTTL(v.ttl)
			if err != nil {
				return fmt.Errorf("invalid TTL: %w", err)
			}
		}
		return s.kv.Set(v.key, v.val, ttl)

	case GetCommand:
		slog.Info("sombody wants to get a key from teh hashtable", "key", v.key)

		val, ok := s.kv.Get(v.key)
		if !ok {
			_, err = msg.peer.Send([]byte("-ERR key not found\r\n"))
		} else {
			_, err = msg.peer.Send(val)
		}
		if err != nil {
			slog.Error("failed to send response", "err", err)
		}

	case DelCommand:
		slog.Info("sombody wants to delete a record from the hashtable", "key", v.key)
		ok := s.kv.Del(v.key)
		if ok {
			_, err = msg.peer.Send([]byte(":1\r\n"))
		} else {
			_, err = msg.peer.Send([]byte(":0\r\n"))
		}
		return err

	case ExistsCommand:
		slog.Info("sombody wants to check if a key exists from the hashtable", "key", v.key)
		ok := s.kv.Exists(v.key)
		if ok {
			_, err = msg.peer.Send([]byte(":1\r\n"))
		} else {
			_, err = msg.peer.Send([]byte(":0\r\n"))
		}
		return err

	case KeysCommand:
		slog.Info("sombody wants to get all the keys from the hashtable")
		keys := s.kv.Keys()
		if len(keys) == 0 {
			_, err = msg.peer.Send([]byte("-ERR key doesn't exists\r\n"))
		} else {
			_, err = msg.peer.Send([]byte(strings.Join(keys, ",")))
		}
		return err

	case TTLCommand:
		slog.Info("sombody wants to check the expiry of the key")
		time_left, check := s.kv.TTL(v.key)

		switch check {
		case -2:
			_, err = msg.peer.Send([]byte("-ERR key doesn't exists"))
		case -1:
			_, err = msg.peer.Send([]byte("-1\r\n"))
		default:
			_, err = msg.peer.Send([]byte(time_left.String()))
		}
		return err
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
