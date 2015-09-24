package internet_dependent_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("GitBuildpack", func() {
	var (
		appName string
	)

	It("uses a buildpack from a git url", func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		Expect(cf.Cf("push", appName, "-m", "128M", "-p", assets.NewAssets().Node, "-b", "https://github.com/cloudfoundry/nodejs-buildpack.git#v1.3.1", "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
})
