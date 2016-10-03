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

var _ = AppsDescribe("Encoding", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-b", Config.GetJavaBuildpackName(),
			"-p", assets.NewAssets().Java,
			"-m", "512M",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("start", appName).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("Does not corrupt UTF-8 characters in filenames", func() {
		var curlResponse string
		Eventually(func() string {
			curlResponse = helpers.CurlApp(Config, appName, "/omega")
			return curlResponse
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("It's Ω!"))
		Expect(curlResponse).To(ContainSubstring("File encoding is UTF-8"))
	})

	Describe("Routing", func() {
		It("Supports URLs with percent-encoded characters", func() {
			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlApp(Config, appName, "/requesturi/%21%7E%5E%24%20%27%28%29?foo=bar+baz%20bing")
				return curlResponse
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("You requested some information about rio rancho properties"))
			Expect(curlResponse).To(ContainSubstring("/requesturi/%21%7E%5E%24%20%27%28%29"))
			Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
		})

		It("transparently proxies both reserved characters and unsafe characters", func() {
			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlApp(Config, appName, "/requesturi/!~^'()$\"?!'()$#!'")
				return curlResponse
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("You requested some information about rio rancho properties"))
			Expect(curlResponse).To(ContainSubstring("/requesturi/!~^'()$\""))
			Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))
		})
	})
})
