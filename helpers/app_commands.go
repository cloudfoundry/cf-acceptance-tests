package helpers

import (
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
	"time"
)

func PushApp(appName string, appPath string) {
	Expect(Cf("push", appName, "-p", appPath)).To(SayWithTimeout("App started", time.Minute*2))
}

func DeleteApp(appName string) {
	Expect(Cf("delete", appName, "-f")).To(SayWithTimeout("OK", time.Minute*2))
}

func AppUri(appName, endpoint string, appsDomain string) string {
	return "http://" + appName + "." + appsDomain + endpoint
}
