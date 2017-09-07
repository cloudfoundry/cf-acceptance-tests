package session

import (
	"io"
	"net/http"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"
)

func StickyHandler(res http.ResponseWriter, req *http.Request) {
	instanceGuid := env.InstanceGuid()

	cookie := http.Cookie{Name: "JSESSIONID", Value: instanceGuid}
	http.SetCookie(res, &cookie)

	io.WriteString(res, "Please read the README.md for help on how to use sticky sessions.")
}
