package helpers

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers/internal"
)

const CURL_TIMEOUT = 30 * time.Second

// Gets an app's endpoint with the specified path
func AppUri(appName, path string) string {
	config := config.LoadConfig()
	uriCreator := &helpersinternal.AppUriCreator{Config: config}

	return uriCreator.AppUri(appName, path)
}

// Curls an app's endpoint and exit successfully before the specified timeout
func CurlAppWithTimeout(appName, path string, timeout time.Duration, args ...string) string {
	appCurler := helpersinternal.NewAppCurler(Curl)
	return appCurler.CurlAndWait(appName, path, timeout, args...)
}

// Curls an app's endpoint and exit successfully before the default timeout
func CurlApp(appName, path string, args ...string) string {
	appCurler := helpersinternal.NewAppCurler(Curl)
	return appCurler.CurlAndWait(appName, path, CURL_TIMEOUT, args...)
}

// Curls an app's root endpoint and exit successfully before the default timeout
func CurlAppRoot(appName string) string {
	appCurler := helpersinternal.NewAppCurler(Curl)
	return appCurler.CurlAndWait(appName, "/", CURL_TIMEOUT)
}

// Returns a function that curls an app's root endpoint and exit successfully before the default timeout
func CurlingAppRoot(appName string) func() string {
	appCurler := helpersinternal.NewAppCurler(Curl)
	return func() string { return appCurler.CurlAndWait(appName, "/", CURL_TIMEOUT) }
}
