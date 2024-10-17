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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = AppsDescribe("Routing Transparency", func() {
	var appName string

	BeforeEach(func() {
		if !Config.GetIncludeHTTP2Routing() {
			Skip(skip_messages.SkipHTTP2RoutingMessage)
		}

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
		curlResponse := helpers.CurlApp(Config, appName, "/requesturi/%21%7E%5E%24%20%27%28%29?foo=bar+baz%20bing")
		Expect(curlResponse).To(ContainSubstring("Request"))

		By("preserving all characters")
		Expect(curlResponse).To(ContainSubstring("/requesturi/%21%7E%5E%24%20%27%28%29"))
		Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
	})

	It("appropriately handles certain reserved/unsafe characters", func() {
		curlResponse := helpers.CurlApp(Config, appName, "/requesturi/!~^'()$\"?!'()$#!'")
		Expect(curlResponse).To(ContainSubstring("Request"))

		By("preserving all characters")
		Expect(curlResponse).To(ContainSubstring("/requesturi/!~^'()$\""))
		Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))
	})
})
