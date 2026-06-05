package main

import (
	"log"
	"log/slog"
	"net"
)

const defaultListenAddr = ":5001"

type Config struct {
	ListenAddr string
}

type Server struct {
	Config
	peers map[*Peer] bool
	ln net.Listener
	addPeerCh chan *Peer
	quitCh chan struct{}
}

func NewServer(cfg Config) *Server{
	if len(cfg.ListenAddr) == 0{
		cfg.ListenAddr = defaultListenAddr
	}
	return &Server{
		Config: cfg,
		peers: make(map[*Peer]bool),
		addPeerCh: make(chan *Peer),
		quitCh: make(chan struct{}),
	}
}

func (s *Server) Start () error{
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil{
		return err
	}
	s.ln = ln
	
	go s.loop()

	slog.Info("server running", "listernAddr", s.ListenAddr)

	return s.acceptLoop()
} 

func (s *Server) loop() {
	for {
		select{
		case peer := <-s.addPeerCh:
			s.peers[peer] = true
		case <-s.quitCh:
			return 
		}
	}
}

func (s *Server) acceptLoop() error{
	for {
		conn, err := s.ln.Accept()
		if err != nil{
			slog.Error("accept errror","err",err)
			continue
		}
		go s.handleCon(conn)
	}
}

func (s *Server) handleCon(conn net.Conn){
	peer := NewPeer(conn)
	s.addPeerCh <- peer
	peer.readLoop()
}

func main() {
	server := NewServer(Config{})
	log.Fatal(server.Start())

}