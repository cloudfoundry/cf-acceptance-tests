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
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = AppsDescribe("An application that's already been pushed", func() {
	var appName string
	var environment *workflowhelpers.Environment

	BeforeEach(func() {
		persistentContext := workflowhelpers.NewPersistentAppContext(config)
		environment = workflowhelpers.NewEnvironment(persistentContext)
		environment.Setup()
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		environment.Teardown()
	})

	BeforeEach(func() {
		appName = config.PersistentAppHost

		appQuery := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
		// might exit with 1 or 0, depending on app status
		output := string(appQuery.Out.Contents())

		if appQuery.ExitCode() == 1 && strings.Contains(output, "not found") {
			pushCommand := cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)
			if pushCommand.ExitCode() != 0 {
				Expect(cf.Cf("delete", "-f", "-r", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Fail("failed to create app")
			}
			app_helpers.SetBackend(appName)
			startCommand := cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)
			if startCommand.ExitCode() != 0 {
				Expect(cf.Cf("delete", "-f", "-r", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Fail("persistent app failed to stage")
			}
		}

		if appQuery.ExitCode() == 0 && strings.Contains(output, "stopped") {
			Expect(cf.Cf("start", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		}
	})

	It("can be restarted and still come up", func() {
		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, CF_PUSH_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		Expect(cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))

		Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, CF_PUSH_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
	})
})
