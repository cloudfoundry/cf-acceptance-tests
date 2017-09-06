package log

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/gorilla/mux"
)

func MakeSpewHandler(w io.Writer) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		kbytes, _ := strconv.Atoi(mux.Vars(req)["kbytes"])

		k := make([]byte, 1024)
		for i := range k {
			k[i] = '1'
		}

		for i := 0; i < kbytes; i++ {
			fmt.Fprintf(w, "%s\n", k)
		}

		io.WriteString(res, fmt.Sprintf("Just wrote %d kbytes to the log", kbytes))
	}
}

func MakeSleepHandler(w io.Writer, clock clock.Clock) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		logSpeed, _ := strconv.Atoi(mux.Vars(req)["logspeed"])

		fmt.Fprintf(w, "Muahaha... let's go. Waiting %f seconds between loglines. Logging 'Muahaha...' every time.\n", float64(logSpeed)/1000000.0)

		sequence := 1
		ticker := clock.NewTicker(time.Duration(logSpeed) * time.Microsecond)
		go func() {
			for {
				select {
				case t := <-ticker.C():
					fmt.Fprintf(w, "Log: %s Muahaha...%d...%s\n", req.Host, sequence, t.Format(time.RFC3339))
					sequence++
				}
			}
		}()
	}
}
