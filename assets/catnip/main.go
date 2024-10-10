package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"code.cloudfoundry.org/clock"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/router"
)

func main() {
	port := os.Getenv("PORT")
	fmt.Printf("listening on port %s...\n", port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router.New(os.Stdout, clock.NewClock()),
	}
	log.Fatal(server.ListenAndServe())
}
