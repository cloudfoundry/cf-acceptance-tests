package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var run bool

const helpString = `Endpoints:

  * /log/sleep/:logspeed - set the pause between loglines to a millionth fraction of a second
  * /log/bytesize/:bytesize - set the size of each logline in bytes
  * /log/stop - stops any running logging
`

func main() {
	run = false

	http.HandleFunc("/log/sleep/", logSpeed)
	http.HandleFunc("/log/bytesize/", logBytesize)
	http.HandleFunc("/log/stop", logStop)
	http.HandleFunc("/", help)
	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func help(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte(helpString))
}

func logSpeed(w http.ResponseWriter, r *http.Request) {
	if run {
		w.WriteHeader(200)
		w.Write([]byte("Already running.  Use /log/stop and then restart."))
	}

	sleepTime, err := strconv.Atoi(strings.Split(r.URL.Path, "/")[3])
	if err != nil {
		panic(err)
	}

	sleepTimeInSeconds := float64(sleepTime) / float64(1000000)

	logline := fmt.Sprintf("Muahaha... let's go. Waiting %f seconds between loglines. Logging 'Muahaha...' every time.\n", sleepTimeInSeconds)
	fmt.Printf(logline)

	run = true

	go func() {
		for run {
			time.Sleep(time.Duration(sleepTime) * time.Microsecond)
			fmt.Println(fmt.Sprintf("Log: %s Muahaha...", r.Host))
		}
	}()

	w.WriteHeader(200)
	w.Write([]byte(logline))
}

func logBytesize(w http.ResponseWriter, r *http.Request) {
	run = true

	byteSize, err := strconv.Atoi(strings.Split(r.URL.Path, "/")[3])
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	for i := 0; i < byteSize; i++ {
		buffer.WriteString("0")
	}
	logString := buffer.String()

	fmt.Println(fmt.Sprintf("Muahaha... let's go. No wait. Logging %d bytes per logline.", byteSize))

	for run {
		fmt.Println(logString)
	}
	w.WriteHeader(200)
}

func logStop(w http.ResponseWriter, r *http.Request) {
	run = false

	logline := fmt.Sprintf("Stopped logs %s\n", time.Now().Format("2006-01-02 15:04:05 -0700"))
	fmt.Println(logline)
	w.WriteHeader(200)
}
