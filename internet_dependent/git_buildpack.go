package internet_dependent_test

import (
	"slices"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var ossStacks = []string{"cflinuxfs4", "cflinuxfs5"}

var _ = InternetDependentDescribe("GitBuildpack", func() {
	var (
		appName string
	)

	It("uses a buildpack from a git url", func() {
		ossStackIndex := slices.IndexFunc(Config.GetStacks(), func(stack string) bool {
			return slices.Contains(ossStacks, stack)
		})
		if ossStackIndex == -1 {
			Skip(skip_messages.SkipNonOSSStackMessage)
		}
		stack := Config.GetStacks()[ossStackIndex]

		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Node,
			"-b", "https://github.com/cloudfoundry/nodejs-buildpack.git#v1.9.3",
			"-s", stack,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Hello from a node app!"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})
})
