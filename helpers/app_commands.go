package helpers

import (
	"github.com/vito/cmdtest"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

func AppUri(appName, endpoint, appsDomain string) string {
	return "http://" + appName + "." + appsDomain + endpoint
}

func Curling(appName, endpoint, appsDomain string) func() *cmdtest.Session {
	return func() *cmdtest.Session {
		return Curl(AppUri(appName, endpoint, appsDomain))
	}
}
