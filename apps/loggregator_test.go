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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("loggregator", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
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

			Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))

			expectedLogMessage := fmt.Sprintf("OUT %s.%s", appName, helpers.LoadConfig().AppsDomain)
			Eventually(logs, (DEFAULT_TIMEOUT + time.Minute)).Should(Say(expectedLogMessage))
		})
	})

	Context("cf logs --recent", func() {
		It("makes loggregator buffer and dump log messages", func() {
			logs := cf.Cf("logs", appName, "--recent").Wait(DEFAULT_TIMEOUT)
			Expect(logs).To(Exit(0))
			Expect(logs).To(Say("Connected, dumping recent logs for app"))
			Expect(logs).To(Say("OUT Created app"))
			Expect(logs).To(Say("OUT Starting app instance"))
		})
	})
})
