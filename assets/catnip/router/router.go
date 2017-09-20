package router

import (
	"io"
	"net/http"

	"code.cloudfoundry.org/clock"
	"github.com/gorilla/mux"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/health"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/linux"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/log"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/session"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/signal"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/text"
)

func New(out io.Writer, clock clock.Clock) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", HomeHandler).Methods(http.MethodGet)
	r.HandleFunc("/id", env.InstanceGuidHandler).Methods(http.MethodGet)
	r.HandleFunc("/myip", linux.MyIPHandler).Methods(http.MethodGet)
	r.HandleFunc("/health", health.HealthHander).Methods(http.MethodGet)
	r.HandleFunc("/session", session.StickyHandler).Methods(http.MethodPost)
	r.HandleFunc("/env.json", env.JSONHandler).Methods(http.MethodGet)
	r.HandleFunc("/env/{name}", env.NameHandler).Methods(http.MethodGet)
	r.HandleFunc("/lsb_release", linux.ReleaseHandler).Methods(http.MethodGet)
	r.HandleFunc("/sigterm/KILL", signal.KillHandler).Methods(http.MethodGet)
	r.HandleFunc("/logspew/{kbytes}", log.MakeSpewHandler(out)).Methods(http.MethodGet)
	r.HandleFunc("/largetext/{kbytes}", text.LargeHandler).Methods(http.MethodGet)
	r.HandleFunc("/log/sleep/{logspeed}", log.MakeSleepHandler(out, clock)).Methods(http.MethodGet)
	r.HandleFunc("/curl/{host}", linux.CurlHandler).Methods(http.MethodGet)
	r.HandleFunc("/curl/{host}/", linux.CurlHandler).Methods(http.MethodGet)
	r.HandleFunc("/curl/{host}/{port}", linux.CurlHandler).Methods(http.MethodGet)

	return r
}

func HomeHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, "Catnip?")
}
