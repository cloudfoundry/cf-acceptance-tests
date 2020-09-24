package apps

import (
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
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

	Context("Kubernetes Istio/Envoy Proxy Behavior", func() {
		BeforeEach(func() {
			if !Config.RunningOnK8s() {
				Skip(skip_messages.SkipVMsMessage)
			}
		})

		It("normalizes URLs with percent-encoded characters by decoding certain characters", func() {
			curlResponse := helpers.CurlApp(Config, appName, "/requesturi/%21%7E%5E%24%20%27%28%29?foo=bar+baz%20bing")
			Expect(curlResponse).To(ContainSubstring("Request"))
			By("decoding unsafe characters like ~")
			Expect(curlResponse).To(ContainSubstring("/requesturi/%21~%5E%24%20%27%28%29"))
			Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
		})

		It("encodes certain reserved/unsafe characters", func() {
			curlResponse := helpers.CurlApp(Config, appName, "/requesturi/!~^'()$\"?!'()$#!'")
			Expect(curlResponse).To(ContainSubstring("Request"))
			By("normalizing unsafe characters such as ^ and \" in the path")
			Expect(curlResponse).To(ContainSubstring("/requesturi/!~%5E'()$%22"))
			By("truncating hash")
			Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))
		})
	})

	Context("Gorouter Behavior", func() {
		BeforeEach(func() {
			if Config.RunningOnK8s() {
				Skip(skip_messages.SkipK8sMessage)
			}
		})

		It("supports URLs with percent-encoded characters", func() {
			curlResponse := helpers.CurlApp(Config, appName, "/requesturi/%21%7E%5E%24%20%27%28%29?foo=bar+baz%20bing")
			Expect(curlResponse).To(ContainSubstring("Request"))
			By("preserving all characters")
			Expect(curlResponse).To(ContainSubstring("/requesturi/%21%7E%5E%24%20%27%28%29"))
			Expect(curlResponse).To(ContainSubstring("Query String is [foo=bar+baz%20bing]"))
		})

		It("transparently proxies both reserved characters and unsafe characters", func() {
			curlResponse := helpers.CurlApp(Config, appName, "/requesturi/!~^'()$\"?!'()$#!'")
			Expect(curlResponse).To(ContainSubstring("Request"))
			By("preserving all characters")
			Expect(curlResponse).To(ContainSubstring("/requesturi/!~^'()$\""))
			By("preserving all characters except hash")
			Expect(curlResponse).To(ContainSubstring("Query String is [!'()$]"))
		})
	})
})
