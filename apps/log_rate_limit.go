package apps

import (
	"regexp"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	logshelper "github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("log rate limit", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Dora,
			"-l", "1K",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("when a log rate limit is defined", func() {
		var logs *Session

		BeforeEach(func() {
			logs = logshelper.Follow(appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt()
			}
		})

		It("enforces the log rate limit", func() {
			helpers.CurlApp(Config, appName, "/logspew/2")
			Eventually(logs).Should(Say(strings.Repeat("1", 1024)))
			Eventually(logs).Should(Say(regexp.QuoteMeta("app instance exceeded log rate limit (1024 bytes/sec)")), "log rate limit not enforced")
			Consistently(logs).ShouldNot(Say("11111"), "logs above the limit were not dropped")

			By("sleeping so that the app is allowed to output more logs")
			time.Sleep(time.Second)

			helpers.CurlApp(Config, appName, "/logspew/2")
			Eventually(logs).Should(Say(strings.Repeat("1", 1024)))
			Eventually(logs).Should(Say(regexp.QuoteMeta("app instance exceeded log rate limit (1024 bytes/sec)")), "log rate limit not enforced")
			Consistently(logs).ShouldNot(Say("11111"), "logs above the limit were not dropped")
		})
	})

})
