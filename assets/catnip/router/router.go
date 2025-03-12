package router

import (
	"io"
	"net/http"

	"code.cloudfoundry.org/clock"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/file"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/health"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/linux"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/log"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/session"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/signal"
	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/text"
)

func New(out io.Writer, clock clock.Clock) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", HomeHandler)
	r.Get("/id", env.InstanceGuidHandler)
	r.Get("/myip", linux.MyIPHandler)
	r.Get("/health", health.HealthHander)
	r.Post("/session", session.StickyHandler)
	r.Get("/env.json", env.JSONHandler)
	r.Get("/env/{name}", env.NameHandler)
	r.Get("/lsb_release", linux.ReleaseHandler)
	r.Get("/sigterm/KILL", signal.KillHandler)
	r.Get("/logspew/{kbytes}", log.MakeSpewHandler(out))
	r.Get("/largetext/{kbytes}", text.LargeHandler)
	r.Get("/log/sleep/{logspeed}", log.MakeSleepHandler(out, clock))
	r.Get("/curl/{host}", linux.CurlHandler)
	r.Get("/curl/{host}/", linux.CurlHandler)
	r.Get("/curl/{host}/{port}", linux.CurlHandler)
	r.Get("/file/{filename}", file.FileHandler)

	return r
}

func HomeHandler(res http.ResponseWriter, req *http.Request) {
	io.WriteString(res, "Catnip?")
}
