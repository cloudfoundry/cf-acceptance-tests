package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/vito/runtime-integration/helpers"
)

var _ = Describe("A running application", func() {
	BeforeEach(func() {
		AppName = RandomName()

		Expect(Cf("push", AppName, "-p", doraPath, "-i", "2")).To(
			SayWithTimeout("Started", 2*time.Minute),
		)
	})

	AfterEach(func() {
		Expect(Cf("delete", AppName, "-f")).To(
			SayWithTimeout("OK", 30*time.Second),
		)
	})

	It("can be queried for state by instance", func() {
		app := Cf("app", AppName)
		Expect(app).To(Say("#0"))
		Expect(app).To(Say("#1"))
	})

	It("can have its files inspected", func() {
		Expect(Cf("files", AppName)).To(Say("app/"))
		Expect(Cf("files", AppName, "app/")).To(Say("config.ru"))
		Expect(Cf("files", AppName, "app/config.ru")).To(
			Say("run Sinatra::Application"),
		)
	})
})
