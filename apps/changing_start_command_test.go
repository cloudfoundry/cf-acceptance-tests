package apps

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("Changing an app's start command", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Context("by using the command flag", func() {
		BeforeEach(func() {
			Expect(cf.Cf(
				"push", appName,
				"--no-start",
				"-b", config.RubyBuildpackName,
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", helpers.LoadConfig().AppsDomain,
				"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
			).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		It("takes effect after a restart, not requiring a push", func() {
			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/FOO")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("foo"))

			guid := cf.Cf("app", appName, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			workflowhelpers.ApiRequest(
				"PUT",
				"/v2/apps/"+appGuid,
				nil,
				DEFAULT_TIMEOUT,
				`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
			)

			Expect(cf.Cf("stop", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/FOO")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("404"))

			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(appName, "/env/FOO")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("bar"))
		})
	})

	Context("by using a Procfile", func() {
		type AppResource struct {
			Entity struct {
				DetectedStartCommand string `json:"detected_start_command"`
			} `json:"entity"`
		}

		type AppsResponse struct {
			Resources []AppResource `json:"resources"`
		}

		BeforeEach(func() {
			Expect(cf.Cf("push", appName, "--no-start", "-b", config.NodejsBuildpackName, "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().NodeWithProcfile, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		})

		It("detects the use of the start command in the 'web' process type", func() {
			var appsResponse AppsResponse
			cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait(DEFAULT_TIMEOUT).Out.Contents()
			json.Unmarshal(cfResponse, &appsResponse)

			Expect(appsResponse.Resources[0].Entity.DetectedStartCommand).To(Equal("node app.js"))
		})
	})
})
