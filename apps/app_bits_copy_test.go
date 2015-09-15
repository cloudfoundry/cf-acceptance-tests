package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Copy app bits", func() {
	var golangAppName string
	var helloWorldAppName string

	BeforeEach(func() {
		golangAppName = generator.PrefixedRandomName("CATS-APP-")
		helloWorldAppName = generator.PrefixedRandomName("CATS-APP-")

		Expect(cf.Cf("push", golangAppName, "-m", "128M", "-p", assets.NewAssets().Golang, "--no-start", "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("push", helloWorldAppName, "-m", "128M", "-p", assets.NewAssets().HelloWorld, "--no-start", "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", golangAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", helloWorldAppName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Copies over the package from the source app to the destination app", func() {
		Expect(cf.Cf("copy-source", helloWorldAppName, golangAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(golangAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
	})
})
