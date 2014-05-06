package helpers

import (
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

const CURL_TIMEOUT = 10 * time.Second

func AppUri(appName, endpoint, appsDomain string) string {
	return "http://" + appName + "." + appsDomain + endpoint
}

// Curl an app's endpoint and exit successfully before the specified timeout
func CurlAppWithTimeout(appName, path string, timeout time.Duration) string {
	appsDomain := LoadConfig().AppsDomain
	url := "http://" + appName + "." + appsDomain + path
	curl := runner.Curl(url).Wait(timeout)
	gomega.Expect(curl).To(gexec.Exit(0))
	return string(curl.Out.Contents())
}

// Curl an app's endpoint and exit successfully before the default timeout
func CurlApp(appName, path string) string {
	return CurlAppWithTimeout(appName, path, CURL_TIMEOUT)
}

// Curl an app's root endpoint and exit successfully before the default timeout
func CurlAppRoot(appName string) string {
	return CurlApp(appName, "/")
}
