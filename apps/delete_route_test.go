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

var _ = Describe("Delete Route", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")

		Expect(cf.Cf("push", appName, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("delete the route", func() {
		It("completes successfully", func() {
			Expect(cf.Cf("delete-route", helpers.LoadConfig().AppsDomain, "-n", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	})
})
