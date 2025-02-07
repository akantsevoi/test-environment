package main

import (
	"fmt"
	"net"

	"github.com/akantsevoi/test-environment/pkg/logger"
)

type TCPServer struct {
	port             int
	incomingMessages chan []byte
}

func NewTCPServer(port int) (*TCPServer, chan []byte) {
	incomingMessages := make(chan []byte)
	return &TCPServer{
		port:             port,
		incomingMessages: incomingMessages,
	}, incomingMessages
}

func (s *TCPServer) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		logger.Fatalf(logger.Network, "Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Errorf(logger.Network, "Failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		logger.Errorf(logger.Network, "Error reading from connection: %v", err)
		return
	}

	logger.Infof(logger.Network, "Received message: %s", string(buffer[:n]))

	s.incomingMessages <- buffer[:n]
}

func (s *TCPServer) SendMessage(targetPod string, message []byte) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s.maroon:8080", targetPod))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", targetPod, err)
	}
	defer conn.Close()

	_, err = conn.Write(message)
	return err
}
