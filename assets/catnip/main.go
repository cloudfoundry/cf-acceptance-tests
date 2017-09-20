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
	fmt.Printf("listening on port %s...\n", os.Getenv("PORT"))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), router.New(os.Stdout, clock.NewClock())))
}
