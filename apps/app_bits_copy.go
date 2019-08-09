package apps

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("Copy app bits", func() {
	var golangAppName string
	var helloWorldAppName string

	BeforeEach(func() {
		golangAppName = random_name.CATSRandomName("APP")
		helloWorldAppName = random_name.CATSRandomName("APP")

		Expect(cf.Push(golangAppName,
			"--no-start",
			"-b", Config.GetRubyBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Golang,
			"-d", Config.GetAppsDomain(),
		).Wait()).To(Exit(0))
		Expect(cf.Push(helloWorldAppName,
			"--no-start",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
			"-d", Config.GetAppsDomain(),
		).Wait()).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(golangAppName)
		app_helpers.AppReport(helloWorldAppName)

		Expect(cf.Cf("delete", golangAppName, "-f", "-r").Wait()).To(Exit(0))
		Expect(cf.Cf("delete", helloWorldAppName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("Copies over the package from the source app to the destination app", func() {
		Expect(cf.Cf("copy-source", helloWorldAppName, golangAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, golangAppName)
		}).Should(ContainSubstring("Hello, world!"))
	})
})
