package routing

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gexec"
)

var _ = RoutingDescribe("Session Affinity", func() {
	var stickyAsset = assets.NewAssets().HelloRouting

	Context("when an app sets a JSESSIONID cookie", func() {
		var (
			appName         string
			cookieStorePath string
		)
		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			Expect(cf.Push(appName,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", stickyAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			cookieStore, err := ioutil.TempFile("", "cats-sticky-session")
			Expect(err).ToNot(HaveOccurred())
			cookieStorePath = cookieStore.Name()
			cookieStore.Close()
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
			err := os.Remove(cookieStorePath)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when an app has multiple instances", func() {
			BeforeEach(func() {
				Expect(cf.Cf("scale", appName, "-i", "3").Wait()).To(Exit(0))
			})

			Context("when the client sends VCAP_ID and JSESSION cookies", func() {
				It("routes every request to the same instance", func() {
					var body string

					Eventually(func() string {
						body = curlAppWithCookies(appName, "/", cookieStorePath)
						return body
					}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", appName)))

					index := parseInstanceIndex(body)

					Consistently(func() string {
						return curlAppWithCookies(appName, "/", cookieStorePath)
					}, 3*time.Second).Should(ContainSubstring(fmt.Sprintf("Hello, %s at index: %d", appName, index)))
				})
			})
		})
	})

	Context("when an app does not set a JSESSIONID cookie", func() {
		var (
			helloWorldAsset = assets.NewAssets().HelloRouting

			appName string
		)

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			Expect(cf.Push(appName,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", helloWorldAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
		})

		Context("when an app has multiple instances", func() {
			BeforeEach(func() {
				Expect(cf.Cf("scale", appName, "-i", "3").Wait()).To(Exit(0))
			})

			Context("when the client does not send VCAP_ID and JSESSION cookies", func() {
				It("routes requests round robin to all instances", func() {
					var body string

					Eventually(func() string {
						body = helpers.CurlAppRoot(Config, appName)
						return body
					}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", appName)))

					indexPre := parseInstanceIndex(body)

					Eventually(func() int {
						body := helpers.CurlAppRoot(Config, appName)
						index := parseInstanceIndex(body)
						return index
					}).ShouldNot(Equal(indexPre))
				})
			})
		})
	})

	Context("when two apps have different context paths", func() {
		var (
			app1Path        = "/app1"
			app2Path        = "/app2"
			app1            string
			app2            string
			hostname        string
			cookieStorePath string
		)

		BeforeEach(func() {
			domain := Config.GetAppsDomain()

			app1 = random_name.CATSRandomName("APP")
			Expect(cf.Push(app1,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", stickyAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			app2 = random_name.CATSRandomName("APP")
			Expect(cf.Push(app2,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", stickyAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("scale", app1, "-i", "3").Wait()).To(Exit(0))
			Expect(cf.Cf("scale", app2, "-i", "3").Wait()).To(Exit(0))
			hostname = random_name.CATSRandomName("ROUTE")

			Expect(cf.Cf("map-route", app1, domain, "--hostname", hostname, "--path", app1Path).Wait()).To(Exit(0))
			Expect(cf.Cf("map-route", app2, domain, "--hostname", hostname, "--path", app2Path).Wait()).To(Exit(0))

			cookieStore, err := ioutil.TempFile("", "cats-sticky-session")
			Expect(err).ToNot(HaveOccurred())
			cookieStorePath = cookieStore.Name()
			cookieStore.Close()
		})

		AfterEach(func() {
			app_helpers.AppReport(app1)
			app_helpers.AppReport(app2)
			Expect(cf.Cf("delete", app1, "-f", "-r").Wait()).To(Exit(0))
			Expect(cf.Cf("delete", app2, "-f", "-r").Wait()).To(Exit(0))

			err := os.Remove(cookieStorePath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Sticky session should work", func() {
			var body string

			// Hit the APP1
			Eventually(func() string {
				body = curlAppWithCookies(hostname, app1Path, cookieStorePath)
				return body
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", app1)))

			index1 := parseInstanceIndex(body)

			// Hit the APP2
			Eventually(func() string {
				body = curlAppWithCookies(hostname, app2Path, cookieStorePath)
				return body
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", app2)))

			index2 := parseInstanceIndex(body)

			// Hit the APP1 again to verify that the session is stick to the right instance.
			Eventually(func() string {
				return curlAppWithCookies(hostname, app1Path, cookieStorePath)
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s at index: %d", app1, index1)))

			// Hit the APP2 again to verify that the session is stick to the right instance.
			Eventually(func() string {
				return curlAppWithCookies(hostname, app2Path, cookieStorePath)
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s at index: %d", app2, index2)))
		})
	})

	Context("when one app has a root path and another with a context path", func() {
		var (
			app2Path        = "/app2"
			app1            string
			app2            string
			hostname        string
			cookieStorePath string
		)

		BeforeEach(func() {
			domain := Config.GetAppsDomain()

			app1 = random_name.CATSRandomName("APP")
			Expect(cf.Push(app1,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", stickyAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			app2 = random_name.CATSRandomName("APP")
			Expect(cf.Push(app2,
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", stickyAsset,
				"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Expect(cf.Cf("scale", app1, "-i", "3").Wait()).To(Exit(0))
			Expect(cf.Cf("scale", app2, "-i", "3").Wait()).To(Exit(0))
			hostname = app1

			Expect(cf.Cf("map-route", app2, domain, "--hostname", hostname, "--path", app2Path).Wait()).To(Exit(0))

			cookieStore, err := ioutil.TempFile("", "cats-sticky-session")
			Expect(err).ToNot(HaveOccurred())
			cookieStorePath = cookieStore.Name()
			cookieStore.Close()
		})

		AfterEach(func() {
			app_helpers.AppReport(app1)
			app_helpers.AppReport(app2)

			Expect(cf.Cf("delete", app1, "-f", "-r").Wait()).To(Exit(0))
			Expect(cf.Cf("delete", app2, "-f", "-r").Wait()).To(Exit(0))

			err := os.Remove(cookieStorePath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Sticky session should work", func() {
			var body string

			// 1: Hit the APP1: the root app. We can set the cookie of the root path.
			// Path: /
			Eventually(func() string {
				body = curlAppWithCookies(hostname, "/", cookieStorePath)
				return body
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", app1)))

			index1 := parseInstanceIndex(body)

			// 2: Hit the APP2. App2 has a path. We can set the cookie of the APP2 path.
			// Path: /app2
			Eventually(func() string {
				body = curlAppWithCookies(hostname, app2Path, cookieStorePath)
				return body
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s", app2)))

			index2 := parseInstanceIndex(body)

			// To do list:
			// 3. Hit the APP1 (root APP) again, to ensure that the instance ID is
			// stick correctly. Only send the first session ID.
			Eventually(func() string {
				return curlAppWithCookies(hostname, "/", cookieStorePath)
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s at index: %d", app1, index1)))

			// 4. Hit the APP2 (path APP) again, to ensure that the instance ID is
			// stick correctly. In this case, both the two cookies will be sent to
			// the server. The curl would store them.
			Eventually(func() string {
				return curlAppWithCookies(hostname, app2Path, cookieStorePath)
			}).Should(ContainSubstring(fmt.Sprintf("Hello, %s at index: %d", app2, index2)))
		})
	})
})

func parseInstanceIndex(body string) int {
	strs := strings.SplitN(body, "index: ", -1)
	indexStr := strings.SplitN(strs[len(strs)-1], "!", -1)
	index, err := strconv.ParseInt(indexStr[0], 10, 0)
	Expect(err).ToNot(HaveOccurred())
	return int(index)
}

func curlAppWithCookies(appName, path string, cookieStorePath string) string {
	uri := helpers.AppUri(appName, path, Config)
	curlCmd := helpers.Curl(Config, uri, "-b", cookieStorePath, "-c", cookieStorePath).Wait(helpers.CURL_TIMEOUT)
	Expect(curlCmd).To(gexec.Exit(0))
	return string(curlCmd.Out.Contents())
}
