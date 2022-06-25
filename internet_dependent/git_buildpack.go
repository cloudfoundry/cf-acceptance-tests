package internet_dependent_test

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-test-helpers/helpers"
)

var _ = InternetDependentDescribe("GitBuildpack", func() {
	var (
		appName string
	)

	SkipOnK8s("URL-based buildpacks not yet supported.")

	It("uses a buildpack from a git url", func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Node,
			"-b", "https://github.com/cloudfoundry/nodejs-buildpack.git#v1.7.24",
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
