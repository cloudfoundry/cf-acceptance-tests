package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/generator"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

var _ = Describe("An application printing a bunch of output", func() {
	var AppName string

	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", TestAssets.Dora)).To(Say("App started"))
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(Say("OK"))
	})

	It("doesn't die when printing 32MB", func() {
		beforeId := string(Curl(AppUri(AppName, "/id", config.AppsDomain)).FullOutput())

		Expect(Curl(AppUri(AppName, "/logspew/33554432", config.AppsDomain))).To(
			Say("Just wrote 33554432 random bytes to the log"),
		)

		// Give time for components (i.e. Warden) to react to the output
		// and potentially make bad decisions (like killing the app)
		time.Sleep(10 * time.Second)

		afterId := string(Curl(AppUri(AppName, "/id", config.AppsDomain)).FullOutput())

		Expect(beforeId).To(Equal(afterId))

		Expect(Curl(AppUri(AppName, "/logspew/2", config.AppsDomain))).To(
			Say("Just wrote 2 random bytes to the log"),
		)
	})
})
