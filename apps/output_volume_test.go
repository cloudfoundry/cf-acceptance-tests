package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var _ = Describe("An application printing a bunch of output", func() {
	var appName string

	BeforeEach(func() {
		appName = RandomName()

		Eventually(Cf("push", appName, "-p", NewAssets().Dora), CFPushTimeout).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(Cf("delete", appName, "-f"), DefaultTimeout).Should(Exit(0))
	})

	It("doesn't die when printing 32MB", func() {
		beforeId := CurlFetcher(appName, "/id", LoadConfig().AppsDomain)()

		logSpew := Curl(AppUri(appName, "/logspew/33554432", LoadConfig().AppsDomain))
		Eventually(logSpew, DefaultTimeout).Should(Say("Just wrote 33554432 random bytes to the log"))

		// Give time for components (i.e. Warden) to react to the output
		// and potentially make bad decisions (like killing the app)
		time.Sleep(10 * time.Second)

		afterId := CurlFetcher(appName, "/id", LoadConfig().AppsDomain)()

		Expect(beforeId).To(Equal(afterId))

		logSpew = Curl(AppUri(appName, "/logspew/2", LoadConfig().AppsDomain))
		Eventually(logSpew, DefaultTimeout).Should(Say("Just wrote 2 random bytes to the log"))
	})
})
