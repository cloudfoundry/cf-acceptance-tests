package apps

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("loggregator", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().LoggregatorLoadGenerator).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Context("cf logs", func() {
		var logs *Session

		BeforeEach(func() {
			logs = cf.Cf("logs", appName)
		})

		AfterEach(func() {
			// logs might be nil if the BeforeEach panics
			if logs != nil {
				logs.Interrupt().Wait(DEFAULT_TIMEOUT)
			}
		})

		It("exercises basic loggregator behavior", func() {
			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say("Connected, tailing logs for app"))

			oneSecond := 1000000 // this app uses millionth of seconds
			Eventually(func() string {
				return helpers.CurlApp(appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Muahaha"))

			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say("Muahaha"))
		})
	})

	Context("cf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			oneSecond := 1000000 // this app uses millionth of seconds
			Eventually(func() string {
				return helpers.CurlApp(appName, fmt.Sprintf("/log/sleep/%d", oneSecond))
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Muahaha"))

			Eventually(func() *Session {
				appLogsSession := cf.Cf("logs", "--recent", appName)
				Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				return appLogsSession
			}, DEFAULT_TIMEOUT).Should(Say("Muahaha"))
		})
	})
})
