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
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("by using the command flag", func() {
		var expectedNullResponse string

		BeforeEach(func() {

			appUrl := "https://" + appName + "." + Config.GetAppsDomain()
			nullSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), appUrl).Wait()
			expectedNullResponse = string(nullSession.Buffer().Contents())

			Expect(cf.Cf(
				"push", appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "FOO=foo ./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("takes effect after a restart, not requiring a push", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}).Should(ContainSubstring("foo"))

			guid := cf.Cf("app", appName, "--guid").Wait().Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			workflowhelpers.ApiRequest(
				"PUT",
				"/v2/apps/"+appGuid,
				nil,
				Config.DefaultTimeoutDuration(),
				`{"command":"FOO=bar ./catnip"}`,
			)

			Expect(cf.Cf("stop", appName).Wait()).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}).Should(ContainSubstring(expectedNullResponse))

			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlApp(Config, appName, "/env/FOO")
			}).Should(ContainSubstring("bar"))
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
			Expect(cf.Cf("push", appName, "-b", Config.GetNodejsBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().NodeWithProcfile).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("detects the use of the start command in the 'web' process type", func() {
			var appsResponse AppsResponse
			cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait().Out.Contents()
			json.Unmarshal(cfResponse, &appsResponse)

			Expect(appsResponse.Resources[0].Entity.DetectedStartCommand).To(Equal("node app.js"))
		})
	})
})
