package main

import (
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
	port := os.Getenv("CF_INSTANCE_PORT")
	for {
		fmt.Printf("ADDRESS: |%s:%s|\n", ip, port)
		time.Sleep(5 * time.Second)
	}
}
