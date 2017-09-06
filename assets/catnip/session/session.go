package session

import (
	"io"
	"net/http"

	"github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"
)

func StickyHandler(res http.ResponseWriter, req *http.Request) {
	instanceId, _ := env.InstanceId()

	cookie := http.Cookie{Name: "JSESSIONID", Value: instanceId}
	http.SetCookie(res, &cookie)

	io.WriteString(res, "Please read the README.md for help on how to use sticky sessions.")
}
