package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"code.cloudfoundry.org/tlsconfig"
)

func main() {
	listenAddress := fmt.Sprintf(":%s", os.Getenv("PORT"))
	var listener net.Listener
	mtls := getCreds()
	if len(mtls.CA) != 0 {
		certPool := x509.NewCertPool()
		appended := certPool.AppendCertsFromPEM([]byte(mtls.CA))
		if !appended {
			panic("cannot append cert to pool")
		}
		cert, err := tls.X509KeyPair([]byte(mtls.Cert), []byte(mtls.Key))
		if err != nil {
			panic(err)
		}
		mtlsConf, err := tlsconfig.Build(
			tlsconfig.WithExternalServiceDefaults(),
			tlsconfig.WithIdentity(cert),
		).Server(
			tlsconfig.WithClientAuthentication(certPool),
		)
		if err != nil {
			panic(err)
		}
		listener, err = tls.Listen("tcp", listenAddress, mtlsConf)
		if err != nil {
			panic(err)
		}
	} else {
		var err error
		listener, err = net.Listen("tcp", listenAddress)
		if err != nil {
			panic(err)
		}
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

type Credentials struct {
	CA   string `json:"ca"`
	Key  string `json:"key"`
	Cert string `json:"cert"`
}

func getCreds() Credentials {
	mtlsString := os.Getenv("MTLS")
	if mtlsString == "" {
		return Credentials{}
	}
	var mtls Credentials
	err := json.Unmarshal([]byte(mtlsString), &mtls)
	if err != nil {
		panic(err)
	}
	return mtls
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
