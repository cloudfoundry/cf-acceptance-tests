package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

var portsFlag = flag.String(
	"ports",
	"8080",
	"Comma delimited list of ports, where the app will be listening to",
)

func main() {
	flag.Parse()
	ports := strings.Split(*portsFlag, ",")

	wg := sync.WaitGroup{}
	for _, port := range ports {
		wg.Add(1)
		go func(wg *sync.WaitGroup, port string) {
			defer wg.Done()

			server := &http.Server{
				Addr: fmt.Sprintf(":%s", port),
				Handler: http.HandlerFunc(func(responseWriter http.ResponseWriter, _ *http.Request) {
					responseWriter.Header().Set("Content-Type", "text/plain")
					responseWriter.Write([]byte(port + "\n"))
				}),
			}
			log.Fatal(server.ListenAndServe())
		}(&wg, port)
	}
	println("Listening on ports ", strings.Join(ports, ", "))
	wg.Wait()
}
