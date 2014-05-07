package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("Delete Route", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()

		Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(helpers.CurlAppRoot(appName)).To(ContainSubstring("Hi, I'm Dora!"))
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
