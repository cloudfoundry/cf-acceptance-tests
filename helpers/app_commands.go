package helpers

import (
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

func AppUri(appName, endpoint, appsDomain string) string {
	return "http://" + appName + "." + appsDomain + endpoint
}

func CurlFetcher(appName, endpoint, appsDomain string) func() string {
	uri := AppUri(appName, endpoint, appsDomain)
	return func() string {
		return string(Curl(uri).Wait(10).Out.Contents())
	}
}
