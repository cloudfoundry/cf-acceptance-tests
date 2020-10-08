package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	go logIP()

	listenAddress := fmt.Sprintf(":%s", os.Getenv("PORT"))
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening for new connections")
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	buffer := make([]byte, 65536)

	for {
		n, err := conn.Read(buffer)

		if err == io.EOF {
			fmt.Println("connection closed")
			return
		} else if err != nil {
			panic(err)
		}

		message := string(buffer[0:n])
		fmt.Println(message)
	}
}

func logIP() {
	ip := os.Getenv("CF_INSTANCE_IP")
	internalIP := os.Getenv("CF_INSTANCE_INTERNAL_IP")
	portsJson := os.Getenv("CF_INSTANCE_PORTS")
	ports := []struct {
		External         uint16 `json:"external"`
		ExternalTLSProxy uint16 `json:"external_tls_proxy"`
	}{}

	err := json.Unmarshal([]byte(portsJson), &ports)
	if err != nil {
		fmt.Printf("Cannot unmarshal CF_INSTANCE_PORTS: %s", err)
		os.Exit(1)
	}

	if len(ports) <= 0 {
		fmt.Printf("CF_INSTANCE_PORTS is empty")
		os.Exit(1)
	}

	port := ports[0].External
	if port == 0 {
		port = ports[0].ExternalTLSProxy
	}

	for {
		fmt.Printf("EXTERNAL ADDRESS: |%s:%d|; INTERNAL ADDRESS: |%s:8080|\n", ip, port, internalIP)
		time.Sleep(5 * time.Second)
	}
}
