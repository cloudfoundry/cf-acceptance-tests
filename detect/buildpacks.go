package detect

import (
	"path/filepath"
	"time"

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

var _ = DetectDescribe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Describe("ruby", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push",
				appName,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().HelloWorld,
				"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, world!"))
		})
	})

	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Node, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-p", assets.NewAssets().Java, "-m", "1024M", "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("start", appName).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Golang, "-f", filepath.Join(assets.NewAssets().Golang, "manifest.yml"), "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("go, world"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Python, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("python, world"))
		})
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout time.Duration

		BeforeEach(func() {
			phpPushTimeout = Config.DetectTimeoutDuration() + 6*time.Minute
		})

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Php, "-d", Config.GetAppsDomain()).Wait(phpPushTimeout)).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello from php"))
		})
	})

	PDescribe("dotnet-core", func() {
		// This test involves a vendored dotnet core app whose locked dotnet version will not be removed
		// from the dotnet core buildpack until end of the version's LTS in 2019

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().DotnetCore, "-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello from dotnet-core"))
		})
	})

	Describe("staticfile", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Staticfile, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello from a staticfile"))
		})
	})

	Describe("binary", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "--no-start", "-b", Config.GetBinaryBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Binary, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hello from a binary"))
		})
	})
})
