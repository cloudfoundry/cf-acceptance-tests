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

func SetBackend(appName string) {
	if Config.GetBackend() == "diego" {
		EnableDiego(appName)
	} else if Config.GetBackend() == "dea" {
		DisableDiego(appName)
	}
}

func EnableDiego(appName string) {
	guid := GetAppGuid(appName)
	Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": true}`), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func DisableDiego(appName string) {
	guid := GetAppGuid(appName)
	Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": false}`), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func AppReport(appName string, timeout time.Duration) {
	if appName == "" {
		return
	}
	Eventually(cf.Cf("app", appName, "--guid"), timeout).Should(Exit())
	Eventually(logs.Tail(Config.GetUseLogCache(), appName), timeout).Should(Exit())
}
