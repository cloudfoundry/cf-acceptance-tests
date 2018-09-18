package main

import (
	"example-apps/proxy/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func launchHandler(port int, downloadHandler, pingHandler, proxyHandler, statsHandler, uploadHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/download/", downloadHandler)
	mux.Handle("/ping/", pingHandler)
	mux.Handle("/proxy/", proxyHandler)
	mux.Handle("/stats", statsHandler)
	mux.Handle("/upload", uploadHandler)
	mux.Handle("/", &handlers.InfoHandler{
		Port: port,
	})
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}

func main() {
	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	stats := &handlers.Stats{
		Latency: []float64{},
	}
	downloadHandler := &handlers.DownloadHandler{}
	pingHandler := &handlers.PingHandler{}
	proxyHandler := &handlers.ProxyHandler{
		Stats: stats,
	}
	statsHandler := &handlers.StatsHandler{
		Stats: stats,
	}
	uploadHandler := &handlers.UploadHandler{}

	launchHandler(systemPort, downloadHandler, pingHandler, proxyHandler, statsHandler, uploadHandler)
}
