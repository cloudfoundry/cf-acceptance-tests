package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers"
)

var _ = Describe("Encoding", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Does not corrupt UTF-8 characters in filenames", func() {
		Expect(cf.Cf("push", appName, "-p", helpers.NewAssets().Java).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		var curlResponse string
		Eventually(func() string {
			curlResponse = helpers.CurlApp(appName, "/omega")
			return curlResponse
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("It's Ω!"))
		Expect(curlResponse).To(ContainSubstring("File encoding is UTF-8"))
	})
})
