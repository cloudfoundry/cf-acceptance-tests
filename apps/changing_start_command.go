package apps

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

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
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	Context("by using the command flag", func() {
		var expectedNullResponse string

		BeforeEach(func() {

			appUrl := "https://" + appName + "." + Config.GetAppsDomain()
			nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait(Config.DefaultTimeoutDuration())
			expectedNullResponse = string(nullSession.Buffer().Contents())

			Expect(cf.Cf(
				"push", appName,
				"--no-start",
				"-b", Config.GetRubyBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Dora,
				"-d", Config.GetAppsDomain(),
				"-c", "FOO=foo bundle exec rackup config.ru -p $PORT",
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("takes effect after a restart, not requiring a push", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("foo"))

			guid := cf.Cf("app", appName, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			workflowhelpers.ApiRequest(
				"PUT",
				"/v2/apps/"+appGuid,
				nil,
				Config.DefaultTimeoutDuration(),
				`{"command":"FOO=bar bundle exec rackup config.ru -p $PORT"}`,
			)

			Expect(cf.Cf("stop", appName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring(expectedNullResponse))

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("bar"))
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
			Expect(cf.Cf("push", appName, "--no-start", "-b", Config.GetNodejsBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().NodeWithProcfile, "-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			app_helpers.SetBackend(appName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("detects the use of the start command in the 'web' process type", func() {
			var appsResponse AppsResponse
			cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			json.Unmarshal(cfResponse, &appsResponse)

			Expect(appsResponse.Resources[0].Entity.DetectedStartCommand).To(Equal("node app.js"))
		})
	})
})
