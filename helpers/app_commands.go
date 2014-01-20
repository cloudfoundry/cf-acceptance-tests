package helpers

import (
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
	"time"
)

func PushApp(appName string, appPath string) {
	Expect(Cf("push", appName, "-p", appPath)).To(SayWithTimeout("App started", time.Minute*1))
}

func DeleteApp(appName string) {
	Expect(Cf("delete", appName, "-f")).To(SayWithTimeout("OK", time.Minute*1))
}
