// +build !noInternet,!noDocker

package docker

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe(deaUnsupportedTag+"Docker Application Lifecycle", func() {
	var appName string

	JustBeforeEach(func() {
		app_helpers.SetBackend(appName)

		By("downloading from dockerhub")
		Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
		Eventually(func() string {
			return helpers.CurlApp(appName, "/env/INSTANCE_INDEX")
		}, DEFAULT_TIMEOUT).Should(Equal("0"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	Describe("running a docker app with a start command", func() {
		BeforeEach(func() {
			appName = generator.RandomName()
			Eventually(cf.Cf(
				"push", appName,
				"--no-start",
				// app is defined by cloudfoundry-incubator/diego-dockerfiles
				"-o", "cloudfoundry/diego-docker-app-custom:latest",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-d", config.AppsDomain,
				"-i", "1",
				"-c", fmt.Sprintf("/myapp/dockerapp -name=%s", appName)),
				DEFAULT_TIMEOUT,
			).Should(Exit(0))
		})

		It("retains its start command through starts and stops", func() {
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(Equal("0"))
			Eventually(helpers.CurlApp(appName, "/name"), DEFAULT_TIMEOUT).Should(Equal(appName))

			By("making the app unreachable when it's stopped")
			Eventually(cf.Cf("stop", appName), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(ContainSubstring("404"))

			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(Equal("0"))
			Eventually(helpers.CurlApp(appName, "/name"), DEFAULT_TIMEOUT).Should(Equal(appName))
		})
	})

	Describe("running a docker app without a start command", func() {
		BeforeEach(func() {
			appName = generator.RandomName()
			Eventually(cf.Cf(
				"push", appName,
				"--no-start",
				// app is defined by cloudfoundry-incubator/diego-dockerfiles
				"-o", "cloudfoundry/diego-docker-app-custom:latest",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-d", config.AppsDomain,
				"-i", "1"),
				DEFAULT_TIMEOUT,
			).Should(Exit(0))
		})

		It("handles docker-defined metadata and environment variables correctly", func() {
			Eventually(helpers.CurlingAppRoot(appName), DEFAULT_TIMEOUT).Should(Equal("0"))

			env_json := helpers.CurlApp(appName, "/env")
			var env_vars map[string]string
			json.Unmarshal([]byte(env_json), &env_vars)

			By("merging garden and docker environment variables correctly")
			// garden set values should win
			Expect(env_vars).To(HaveKey("HOME"))
			Expect(env_vars).NotTo(HaveKeyWithValue("HOME", "/home/some_docker_user"))
			Expect(env_vars).To(HaveKey("VCAP_APPLICATION"))
			Expect(env_vars).NotTo(HaveKeyWithValue("VCAP_APPLICATION", "{}"))
			// docker image values should remain
			Expect(env_vars).To(HaveKeyWithValue("SOME_VAR", "some_docker_value"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_QUOTE", "'"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_SHELL", "$1"))
			// values tested here are defined by:
			// cloudfoundry-incubator/diego-dockerfiles/diego-docker-custom-app/Dockerfile
		})
	})
})
