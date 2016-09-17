package apps

import (
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

		Expect(cf.Cf("push", golangAppName,
			"--no-start",
			"-b", config.RubyBuildpackName,
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Golang,
			"-d", config.AppsDomain,
		).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("push", helloWorldAppName,
			"--no-start",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().HelloWorld,
			"-d", config.AppsDomain,
		).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(golangAppName, DEFAULT_TIMEOUT)
		app_helpers.AppReport(helloWorldAppName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", golangAppName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", helloWorldAppName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Copies over the package from the source app to the destination app", func() {
		app_helpers.SetBackend(golangAppName)
		Expect(cf.Cf("copy-source", helloWorldAppName, golangAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(golangAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
	})
})
