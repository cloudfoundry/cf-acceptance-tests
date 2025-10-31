package detect

import (
	"fmt"
	"path/filepath"
	"time"

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

var _ = DetectDescribe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Describe("ruby", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push",
						appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().HelloWorld,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello, world!"))
				})
			})
		}
	})

	Describe("node", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push",
						appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Node,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from a node app!"))
				})
			})
		}
	})

	Describe("java", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-p", assets.NewAssets().Java,
						"-m", "1024M",
						"-f", filepath.Join(assets.NewAssets().Java, "manifest.yml"),
						"-s", stack,
					).Wait(CF_JAVA_TIMEOUT)).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
				})
			})
		}
	})

	Describe("golang", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Golang,
						"-f", filepath.Join(assets.NewAssets().Golang, "manifest.yml"),
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("go, world"))
				})
			})
		}
	})

	Describe("python", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Python,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("python, world"))
				})
			})
		}
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout time.Duration

		BeforeEach(func() {
			phpPushTimeout = Config.DetectTimeoutDuration() + 6*time.Minute
		})

		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Php,
						"-s", stack,
					).Wait(phpPushTimeout)).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from php"))
				})
			})
		}
	})

	Describe("dotnet-core", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					assetPath, ok := assets.NewAssets().DotnetCore[stack]
					Expect(ok).To(BeTrue(), fmt.Sprintf("dotnet-core app is missing asset for %s stack", stack))
					Expect(cf.Cf("push", appName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assetPath, "-s", stack).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from dotnet-core"))
				})
			})
		}
	})

	Describe("staticfile", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Staticfile,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from a staticfile"))
				})
			})
		}
	})

	Describe("binary", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-b", Config.GetBinaryBuildpackName(),
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Binary,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello from a binary"))
				})
			})
		}
	})

	Describe("nginx", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", DEFAULT_MEMORY_LIMIT,
						"-p", assets.NewAssets().Nginx,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello NGINX!"))
				})
			})
		}
	})

	Describe("r", func() {
		for _, stack := range Config.GetStacks() {
			stack := stack
			Context(fmt.Sprintf("when using %s stack", stack), func() {
				It("makes the app reachable via its bound route", func() {
					Expect(cf.Cf("push", appName,
						"-m", "2G",
						"-p", assets.NewAssets().R,
						"-s", stack,
					).Wait(Config.DetectTimeoutDuration())).To(Exit(0))

					Eventually(func() string {
						return helpers.CurlAppRoot(Config, appName)
					}).Should(ContainSubstring("Hello R!"))
				})
			})
		}
	})
})
