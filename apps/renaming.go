package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("renaming", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf(app_helpers.CatnipWithArgs(
			appName,
			"-m", DEFAULT_MEMORY_LIMIT)...,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		helpers.CurlApp(Config, appName, "log/sleep/1")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("changes the app name in emitted logs without a restart", func() {
		newAppName := random_name.CATSRandomName("APP")
		Expect(cf.Cf("rename", appName, newAppName).Wait()).To(Exit(0))
		appName = newAppName

		appGuid := app_helpers.GetAppGuid(newAppName)
		token := v3_helpers.GetAuthToken()
		Eventually(func() bool {
			resp := logs.RecentEnvelopes(appGuid, token, Config)
			for _, e := range resp.Envelopes.Batch {
				if e.Tags["origin"] == "rep" && e.Tags["app_name"] == newAppName {
					return true
				}
			}
			return false
		}).Should(BeTrue())
	})

})
