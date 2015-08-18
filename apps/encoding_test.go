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

var _ = Describe("Encoding", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.PrefixedRandomName("CATS-APP-")
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Java, "--no-start", "-m", "512M", "-d", config.AppsDomain).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("start", appName).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Does not corrupt UTF-8 characters in filenames", func() {
		var curlResponse string
		Eventually(func() string {
			curlResponse = helpers.CurlApp(appName, "/omega")
			return curlResponse
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("It's Ω!"))
		Expect(curlResponse).To(ContainSubstring("File encoding is UTF-8"))
	})

	Describe("Routing", func() {
		It("Supports URLs with percent-encoded characters", func() {
			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlApp(appName, "/requesturi/%21%7E%5E%24%20%27%28%29?foo=bar+baz%20bing")
				return curlResponse
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("You requested some information about rio rancho properties"))
			Expect(curlResponse).To(ContainSubstring("/requesturi/%21%7E%5E%24%20%27%28%29"))
			Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
		})

		It("transparently proxies both reserved characters and unsafe characters", func() {
			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlApp(appName, "/requesturi/!~^'()$\"?!'()$#!'")
				return curlResponse
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("You requested some information about rio rancho properties"))
			Expect(curlResponse).To(ContainSubstring("/requesturi/!~^'()$\""))
			Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))
		})
	})
})
