package diego

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const (
	DOCKER_IMAGE_DOWNLOAD_DEFAULT_TIMEOUT = 300 * time.Second
)

type SpaceJson struct {
	Resources []struct {
		Metadata struct {
			Guid string
		}
	}
}

var createDockerAppPayload string

var _ = Describe("Docker Application Lifecycle", func() {
	appName := generator.RandomName()
	domain := helpers.LoadConfig().AppsDomain

	BeforeEach(func() {
		createDockerAppPayload = `{"name": "%s", 
								   "memory":512, 
								   "instances":1, 
								   "disk_quota":1024, 
								   "space_guid":"%s", 
								   "docker_image":"cloudfoundry/inigodockertest:latest", 
								   "command":"/dockerapp"}`

	})

	JustBeforeEach(func() {
		url := fmt.Sprintf("/v2/spaces?q=name%%3A%s", context.RegularUserContext().Space)
		jsonResults := SpaceJson{}
		curl := cf.Cf("curl", url).Wait(DEFAULT_TIMEOUT)
		Expect(curl).To(Exit(0))

		json.Unmarshal(curl.Out.Contents(), &jsonResults)
		spaceGuid := jsonResults.Resources[0].Metadata.Guid
		Expect(spaceGuid).NotTo(Equal(""))

		payload := fmt.Sprintf(createDockerAppPayload, appName, spaceGuid)
		Eventually(cf.Cf("curl", "/v2/apps", "-X", "POST", "-d", payload), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("set-env", appName, "CF_DIEGO_RUN_BETA", "true"), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("create-route", context.RegularUserContext().Space, domain, "-n", appName), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("map-route", appName, domain, "-n", appName), DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(cf.Cf("start", appName), DOCKER_IMAGE_DOWNLOAD_DEFAULT_TIMEOUT).Should(Exit(0))
		Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
	})

	Describe("running the app", func() {
		It("merges the warden and docker environment variables", func() {
			env_json := helpers.CurlApp(appName, "/env")
			var env_vars map[string]string
			json.Unmarshal([]byte(env_json), &env_vars)

			// warden set values should win
			Ω(env_vars).Should(HaveKey("HOME"))
			Ω(env_vars).ShouldNot(HaveKeyWithValue("HOME", "/home/some_docker_user"))
			Ω(env_vars).Should(HaveKey("VCAP_APPLICATION"))
			Ω(env_vars).ShouldNot(HaveKeyWithValue("VCAP_APPLICATION", "{}"))
			// docker image values should remain
			Ω(env_vars).Should(HaveKeyWithValue("SOME_VAR", "some_docker_value"))
			Ω(env_vars).Should(HaveKeyWithValue("BAD_QUOTE", "'"))
			Ω(env_vars).Should(HaveKeyWithValue("BAD_SHELL", "$1"))

		})
	})

	Describe("running a docker app without a start command ", func() {
		BeforeEach(func() {
			createDockerAppPayload = `{"name": "%s", 
									   "memory":512, 
									   "instances":1, 
									   "disk_quota":1024, 
									   "space_guid":"%s", 
									   "docker_image":"cloudfoundry/inigodockertest:latest"}`
		})

		It("locates and invokes the start command", func() {
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))

		})
	})

	Describe("stopping an app", func() {
		It("makes the app unreachable while it is stopped", func() {
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))

			Eventually(cf.Cf("stop", appName), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("404"))

			Eventually(cf.Cf("start", appName), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(appName)).Should(Equal("0"))
		})
	})

	Describe("scaling the app", func() {
		JustBeforeEach(func() {
			Eventually(cf.Cf("stop", appName), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("scale", appName, "-i", "3"), DEFAULT_TIMEOUT).Should(Exit(0))
			Eventually(cf.Cf("start", appName), DEFAULT_TIMEOUT).Should(Exit(0))
		})

		It("Retrieves instance information for cf app", func() {
			app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: [0-3]/3"))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
			Expect(app).To(Say("#2"))
			Expect(app).ToNot(Say("#3"))
		})

		It("Retrieves instance information for cf apps", func() {
			app := cf.Cf("apps").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say(appName))
			Expect(app).To(Say("[0-3]/3"))
		})
	})

	Describe("deleting", func() {
		JustBeforeEach(func() {
			Eventually(cf.Cf("delete", appName, "-f"), DEFAULT_TIMEOUT).Should(Exit(0))
		})

		It("removes the application and makes the app unreachable", func() {
			Eventually(cf.Cf("app", appName), DEFAULT_TIMEOUT).Should(Say("not found"))
			Eventually(helpers.CurlingAppRoot(appName)).Should(ContainSubstring("404"))
		})
	})
})
