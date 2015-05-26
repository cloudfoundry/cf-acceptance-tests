package apps

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Node, "-c", "node app.js").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("java", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Java, "--no-start", "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Golang).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Python).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
		})
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout = CF_PUSH_TIMEOUT + 6*time.Minute

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Php).Wait(phpPushTimeout)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
		})
	})

	Describe("staticfile", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Staticfile).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a staticfile"))
		})
	})

	Describe("binary", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-b", "binary_buildpack", "-p", assets.NewAssets().Binary).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a binary"))
		})
	})

	Context("lucid64", func() {
		Describe("node", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Node, "-c", "node app.js").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
			})
		})

		Describe("java", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Java, "--no-start", "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("set-env", appName, "JAVA_OPTS", "-Djava.security.egd=file:///dev/urandom").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))
			})
		})

		Describe("golang", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Golang).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
			})
		})

		Describe("python", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Python).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
			})
		})

		Describe("php", func() {
			// This test requires more time during push, because the php buildpack is slower than your average bear
			var phpPushTimeout = CF_PUSH_TIMEOUT + 6*time.Minute

			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Php).Wait(phpPushTimeout)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
			})
		})

		Describe("staticfile", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-p", assets.NewAssets().Staticfile).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a staticfile"))
			})
		})

		Describe("binary", func() {
			It("makes the app reachable via its bound route", func() {
				Expect(cf.Cf("push", appName, "-s", "lucid64", "-b", "binary_buildpack", "-p", assets.NewAssets().Binary).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

				Eventually(func() string {
					return helpers.CurlAppRoot(appName)
				}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a binary"))
			})
		})
	})
})
