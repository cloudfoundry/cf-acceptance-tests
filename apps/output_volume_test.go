package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/vito/runtime-integration/helpers"
)

var _ = Describe("An application printing a bunch of output", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", doraPath)).To(
			SayWithTimeout("Started", 2*time.Minute),
		)
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(
			SayWithTimeout("OK", 30*time.Second),
		)
	})

	It("doesn't die when printing 32MB", func() {
		beforeId := Curl(AppUri("/id")).FullOutput()

		Expect(Curl(AppUri("/logspew/33554432"))).To(
			SayWithTimeout(
				"Just wrote 33554432 random bytes to the log",
				30 * time.Second,
			),
		)

		// Give time for components (i.e. Warden) to react to the output
		// and potentially make bad decisions (like killing the app)
		time.Sleep(10 * time.Second)

		afterId := Curl(AppUri("/id")).FullOutput()

		Expect(beforeId).To(Equal(afterId))

		Expect(Curl(AppUri("/logspew/2"))).To(
			Say("Just wrote 2 random bytes to the log"),
		)
	})
})

