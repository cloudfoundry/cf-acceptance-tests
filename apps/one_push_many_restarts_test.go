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
	"time"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var _ = Describe("An application that's already been pushed", func() {
	var appName string
	config := LoadConfig()
	var environment *Environment

	BeforeEach(func() {
		persistentContext := NewPersistentAppContext(config)
		environment = NewEnvironment(persistentContext)
		environment.Setup()
	})

	AfterEach(func() {
		environment.Teardown()
	})

	BeforeEach(func() {
		appName = config.PersistentAppHost

		appQuery := Cf("app", appName)

		select {
		case <-appQuery.Out.Detect("not found"):
			Eventually(Cf("push", appName, "-p", NewAssets().Dora), CFPushTimeout).Should(Exit(0))
		case <-appQuery.Out.Detect("running"):
		case <-time.After(DefaultTimeout * time.Second):
			Fail("failed to find or setup app")
		}

		appQuery.Out.CancelDetects()
	})

	It("can be restarted and still come up", func() {
		Eventually(CurlFetcher(appName, "/", config.AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))

		Eventually(Cf("stop", appName), DefaultTimeout).Should(Exit(0))

		Eventually(CurlFetcher(appName, "/", config.AppsDomain), DefaultTimeout).Should(ContainSubstring("404"))

		Eventually(Cf("start", appName), DefaultTimeout).Should(Exit(0))

		Eventually(CurlFetcher(appName, "/", config.AppsDomain), DefaultTimeout).Should(ContainSubstring("Hi, I'm Dora!"))
	})
})
