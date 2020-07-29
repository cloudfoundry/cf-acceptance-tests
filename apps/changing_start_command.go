package apps

import (
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = Describe("Changing an app's start command", func() {
	type AppProcessResponse struct{
		Resources []struct {
			Command string
			Guid string
		}
	}
	type ProcessResponse struct{
		Command string
		Guid string
	}
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("by using the command flag", func() {
		BeforeEach(func() {
			Expect(cf.Cf(
				"push", appName,
				"-b", Config.GetBinaryBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().Catnip,
				"-c", "FOO=foo ./catnip",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		It("takes effect after a restart, not requiring a push", func() {
			guid := cf.Cf("app", appName, "--guid").Wait().Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			var appProcessResponse = AppProcessResponse{}
			workflowhelpers.ApiRequest(
				"GET",
				"/v3/apps/"+appGuid+"/processes?types=web",
				&appProcessResponse,
				Config.DefaultTimeoutDuration(),
			)
			processGuid := appProcessResponse.Resources[0].Guid

			processResponse := ProcessResponse{}
			workflowhelpers.ApiRequest(
				"GET",
				"/v3/processes/"+ processGuid,
				&processResponse,
				Config.DefaultTimeoutDuration(),
				)

			Expect(processResponse.Command).To(Equal("FOO=foo ./catnip"))
			workflowhelpers.ApiRequest(
				"PATCH",
				"/v3/processes/"+processGuid,
				nil,
				Config.DefaultTimeoutDuration(),
				`{"command":"FOO=bar ./catnip"}`,
			)

			Expect(cf.Cf("stop", appName).Wait()).To(Exit(0))
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			workflowhelpers.ApiRequest(
				"GET",
				"/v3/processes/"+processGuid,
				&processResponse,
				Config.DefaultTimeoutDuration(),
			)
			Expect(processResponse.Command).To(Equal("FOO=bar ./catnip"))
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
			guid := cf.Cf("app", appName, "--guid").Wait().Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			var appProcessResponse = AppProcessResponse{}
			workflowhelpers.ApiRequest(
				"GET",
				"/v3/apps/"+appGuid+"/processes?types=web",
				&appProcessResponse,
				Config.DefaultTimeoutDuration(),
			)
			processGuid := appProcessResponse.Resources[0].Guid

			processResponse := ProcessResponse{}
			workflowhelpers.ApiRequest(
				"GET",
				"/v3/processes/"+ processGuid,
				&processResponse,
				Config.DefaultTimeoutDuration(),
			)
			Expect(processResponse.Command).To(Equal("node app.js"))
		})
	})
})
