package main

import (
	"fmt"
	"net"

	"github.com/akantsevoi/test-environment/pkg/logger"
)

type TCPServer struct {
	port int
}

func NewTCPServer(port int) *TCPServer {
	return &TCPServer{
		port: port,
	}
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

	message := string(buffer[:n])
	logger.Infof(logger.Network, "Received message: %s", message)
}

func (s *TCPServer) SendMessage(targetPod string, message string) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s.maroon:8080", targetPod))
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", targetPod, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	return err
}
