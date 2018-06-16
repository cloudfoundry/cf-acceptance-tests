package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/uaago"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	log := log.New(os.Stderr, "", 0)

	if len(args[1:]) != 3 {
		log.Fatalf("Usage %s [URL] [CLIENT_ID] [EXISTING_REFRESH_TOKEN]", args[0])
		return 1
	}

	uaa, err := uaago.NewClient(args[1])
	if err != nil {
		log.Fatalf("Failed to create client: %s", err.Error())
		return 1
	}

	refreshToken, accessToken, err := uaa.GetRefreshToken(args[2], args[3], true)
	if err != nil {
		if refreshToken == "" {
			log.Fatalf("Failed to get new refresh token: %s", err.Error())
		}
		if accessToken == "" {
			log.Fatalf("Failed to get access token: %s", err.Error())
		}
		return 1
	}

	fmt.Printf("REFRESH_TOKEN: %s\n", refreshToken)
	fmt.Printf("ACCESS_TOKEN: %s\n", accessToken)
	return 0
}
