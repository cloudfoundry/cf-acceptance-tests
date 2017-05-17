package backend_compatibility

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

var _ = BackendCompatibilityDescribe("Backend Compatibility", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		Eventually(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Dora,
			"--no-start",
			"-m", DEFAULT_MEMORY_LIMIT,
			"-b", Config.GetRubyBuildpackName(),
			"-d", Config.GetAppsDomain()),
			Config.CfPushTimeoutDuration()).Should(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
		Eventually(cf.Cf("delete", appName, "-f"), Config.DefaultTimeoutDuration()).Should(Exit(0))
	})

	Describe("An app staged on the DEA", func() {
		BeforeEach(func() {
			guid := cf.Cf("app", appName, "--guid").Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			appGuid := strings.TrimSpace(string(guid))

			By("Uploading a droplet staged on the DEA")
			dropletPath := assets.NewAssets().DoraDroplet

			token := v3_helpers.GetAuthToken()
			uploadUrl := fmt.Sprintf("%s%s/v2/apps/%s/droplet/upload", Config.Protocol(), Config.GetApiEndpoint(), appGuid)
			bits := fmt.Sprintf(`droplet=@%s`, dropletPath)
			curl := helpers.Curl(Config, "-v", uploadUrl, "-X", "PUT", "-F", bits, "-H", fmt.Sprintf("Authorization: %s", token)).Wait(Config.DefaultTimeoutDuration())
			Expect(curl).To(Exit(0))

			var job struct {
				Metadata struct {
					Url string `json:"url"`
				} `json:"metadata"`
			}
			bytes := curl.Out.Contents()
			json.Unmarshal(bytes, &job)
			pollingUrl := job.Metadata.Url

			Eventually(func() *Session {
				return cf.Cf("curl", pollingUrl).Wait(Config.DefaultTimeoutDuration())
			}, Config.DefaultTimeoutDuration()).Should(gbytes.Say("finished"))
		})

		It("runs on Diego", func() {
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("Hi, I'm Dora!"))
		})
	})
})
