package main

import (
	"flag"
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

			log.Fatal(http.ListenAndServe(":"+port, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(port + "\n"))
			})))
		}(&wg, port)
	}
	println("Listening on ports ", strings.Join(ports, ", "))
	wg.Wait()
}
