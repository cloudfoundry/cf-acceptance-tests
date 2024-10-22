package apps

import (
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("Routing Transparency", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetGoBuildpackName(),
			"-p", assets.NewAssets().Golang,
			"-f", filepath.Join(assets.NewAssets().Golang, "manifest.yml"),
			"-m", DEFAULT_MEMORY_LIMIT,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("appropriately handles URLs with percent-encoded characters", func() {
		curlResponse := helpers.CurlApp(Config, appName, "/requesturi/%20%21%22%23%24%25%27%28%29%3C%3E%5E%7C%7E?foo=bar+baz%20bing")
		Expect(curlResponse).To(ContainSubstring("Request"))

		By("preserving all characters")
		Expect(curlResponse).To(ContainSubstring("/requesturi/%20%21%22%23%24%25%27%28%29%3C%3E%5E%7C%7E"))
		Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
	})

	It("appropriately handles certain reserved/unsafe characters", func() {
		curlResponse := helpers.CurlApp(Config, appName, "/requesturi/!~^'()$?!'()$#!'")
		Expect(curlResponse).To(ContainSubstring("Request"))

		By("preserving all characters")
		Expect(curlResponse).To(ContainSubstring("/requesturi/!~^'()$"))
		Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))

		By("preserving double quotes (not for HTTP/2)")
		if !Config.GetIncludeHTTP2Routing() {
			curlResponse = helpers.CurlApp(Config, appName, "/requesturi/\"")
			Expect(curlResponse).To(ContainSubstring("/requesturi/\""))
		}
	})
})
