package app_helpers

import (
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func GetAppGuid(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Eventually(cfApp, Config.DefaultTimeoutDuration()).Should(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func AppReport(appName string, timeout time.Duration) {
	if appName == "" {
		return
	}
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(Exit())
	Eventually(logs.Tail(Config.GetUseLogCache(), appName), timeout).Should(Exit())
}
