package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/uaago"
)

func main() {
	log := log.New(os.Stderr, "", 0)

	if len(os.Args[1:]) != 3 {
		log.Fatalf("Usage %s [URL] [USERNAME] [PASS]", os.Args[0])
	}

	uaa, err := uaago.NewClient(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %s", err.Error())
	}

	token, err := uaa.GetAuthToken(os.Args[2], os.Args[3], false)
	if err != nil {
		log.Fatalf("Faild to get auth token: %s", err.Error())
	}

	fmt.Println(token)
}
