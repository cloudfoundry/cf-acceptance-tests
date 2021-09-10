package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	protobuff "github.com/cloudfoundry/cf-acceptance-tests/assets/grpc/test"
	"google.golang.org/grpc"
)

type server struct {
	protobuff.UnimplementedTestServer
}

func (s *server) Run(c context.Context, r *protobuff.Request) (*protobuff.Response, error) {
	return &protobuff.Response{Body: "Hello"}, nil
}

func main() {
	port := os.Getenv("PORT")
	address := fmt.Sprintf("0.0.0.0:%s", port)
	fmt.Printf("Listening [%s]...\n", address)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error TCP listen: %v", err)
	}

	s := grpc.NewServer()
	protobuff.RegisterTestServer(s, &server{})
	err = s.Serve(listener)
	if err != nil {
		log.Fatalf("Error server serve: %v", err)
	}
}
