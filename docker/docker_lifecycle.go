// +build !noInternet,!noDocker

package docker

import (
	"encoding/json"
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = DockerDescribe("Docker Application Lifecycle", func() {
	var appName string

	JustBeforeEach(func() {
		By("downloading from dockerhub (starting the app)")
		Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/env/INSTANCE_INDEX")
		}).Should(Equal("0"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	PDescribe("running a docker app with a start command", func() {
		var expectedNullResponse string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			appUrl := "https://" + appName + "." + Config.GetAppsDomain()
			nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
			expectedNullResponse = string(nullSession.Buffer().Contents())
			Eventually(cf.Cf(
				"push", appName,
				"--no-start",
				// app is defined by cloudfoundry-incubator/diego-dockerfiles
				"-o", Config.GetPublicDockerAppImage(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-i", "1",
				"-c", fmt.Sprintf("/myapp/dockerapp -name=%s", appName)),
			).Should(Exit(0))
		})

		It("retains its start command through starts and stops", func() {
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("0"))
			Eventually(helpers.CurlApp(Config, appName, "/name")).Should(Equal(appName))

			By("making the app unreachable when it's stopped")
			Eventually(cf.Cf("stop", appName)).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring(expectedNullResponse))

			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("0"))
			Eventually(helpers.CurlApp(Config, appName, "/name")).Should(Equal(appName))
		})
	})

	PDescribe("running a docker app without a start command", func() {
		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			Eventually(cf.Cf(
				"push", appName,
				"--no-start",
				// app is defined by cloudfoundry-incubator/diego-dockerfiles
				"-o", Config.GetPublicDockerAppImage(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-i", "1"),
			).Should(Exit(0))
		})

		It("handles docker-defined metadata and environment variables correctly", func() {
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("0"))

			env_json := helpers.CurlApp(Config, appName, "/env")
			var env_vars map[string]string
			json.Unmarshal([]byte(env_json), &env_vars)

			By("merging garden and docker environment variables correctly")
			// values tested here are defined by:
			// cloudfoundry-incubator/diego-dockerfiles/diego-docker-custom-app/Dockerfile

			// garden set values should win
			Expect(env_vars).To(HaveKey("VCAP_APPLICATION"))
			Expect(env_vars).NotTo(HaveKeyWithValue("VCAP_APPLICATION", "{}"))
			Expect(env_vars).NotTo(HaveKey("TMPDIR"))

			// docker image values should remain
			Expect(env_vars).To(HaveKeyWithValue("HOME", "/home/dockeruser"))
			Expect(env_vars).To(HaveKeyWithValue("SOME_VAR", "some_docker_value"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_QUOTE", "'"))
			Expect(env_vars).To(HaveKeyWithValue("BAD_SHELL", "$1"))
		})

		Context("when env vars are set with 'cf set-env'", func() {
			BeforeEach(func() {
				Eventually(cf.Cf("set-env", appName, "HOME", "/tmp/fakehome")).Should(Exit(0))
				Eventually(cf.Cf("set-env", appName, "TMPDIR", "/tmp/dir")).Should(Exit(0))
			})

			It("prefers the env vars from cf set-env over those in the Dockerfile", func() {
				Eventually(helpers.CurlingAppRoot(Config, appName)).Should(Equal("0"))

				envJson := helpers.CurlApp(Config, appName, "/env")
				var envVars map[string]string
				json.Unmarshal([]byte(envJson), &envVars)

				Expect(envVars).To(HaveKeyWithValue("HOME", "/tmp/fakehome"))
				Expect(envVars).To(HaveKeyWithValue("TMPDIR", "/tmp/dir"))
			})
		})
	})
})
