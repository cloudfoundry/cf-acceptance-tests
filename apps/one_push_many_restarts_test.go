// This is a defensive test against the CC no longer knowing how to find an
// existing app's bits. This can happen if the scheme of the app's paths in
// the blobstore changes without being backwards-compatible.
//
// If this is not caught before a deploy, all running apps will go down, as
// during evacuation of the DEAs, the CC will not know to look in their old
// path format in the blob store.
//
// This tests pushes the app once (checking if it already exists), and then
// just restarts it on later runs.

package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var _ = Describe("An application that's already been pushed", func() {
	var AppName string

	BeforeEach(func() {
		AppName = config.PersistentAppHost
		Expect(Cf("target", "-s", "persistent-space")).To(ExitWith(0))

		Expect(Cf("app", AppName)).To(SayBranches(
			cmdtest.ExpectBranch{
				"not found",
				func() {
					Expect(
						Cf("push", AppName, "-p", TestAssets.Dora),
					).To(Say("App started"))
				},
			},
			cmdtest.ExpectBranch{
				"Showing health and status",
				func() {
				},
			},
		))
	})

	It("can be restarted and still come up", func() {
		Eventually(Curling(AppName, "/", config.AppsDomain)).Should(Say("Hi, I'm Dora!"))

		Expect(Cf("stop", AppName)).To(Say("OK"))

		Eventually(Curling(AppName, "/", config.AppsDomain)).Should(Say("404"))

		Expect(Cf("start", AppName)).To(Say("App started"))

		Eventually(Curling(AppName, "/", config.AppsDomain)).Should(Say("Hi, I'm Dora!"))
	})
})
