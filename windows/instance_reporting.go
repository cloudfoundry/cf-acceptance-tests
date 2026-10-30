package windows

import (
	"fmt"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const EXCEED_CELL_MEMORY = "900g"

var _ = WindowsDescribe("Getting instance information", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-p", assets.NewAssets().WindowsWebapp,
			"-c", ".\\webapp.exe",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
	})

	Context("scaling memory", func() {
		BeforeEach(func() {
			setTotalMemoryLimit(RUNAWAY_QUOTA_MEM_LIMIT)
		})

		AfterEach(func() {
			setTotalMemoryLimit("10G")
		})

		It("fails when scaled beyond available resources", func() {
			scale := cf.Cf("scale", appName, "-m", EXCEED_CELL_MEMORY, "-f")
			Eventually(scale, Config.CfPushTimeoutDuration()).Should(Or(Say("insufficient resources"), Say("down")))
			scale.Kill()

			app := cf.Cf("app", appName)
			Eventually(app).Should(Exit(0))
			Expect(app.Out).NotTo(Say("instances: 1/1"))
		})
	})
})

func setTotalMemoryLimit(memoryLimit string) {
	var org struct {
		Resources []struct {
			Relationships struct {
				Quota struct {
					Data struct {
						GUID string `json:"guid"`
					} `json:"data"`
				} `json:"quota"`
			} `json:"relationships"`
		} `json:"resources"`
	}
	orgEndpoint := fmt.Sprintf("/v3/organizations?names=%s", TestSetup.GetOrganizationName())
	ApiRequest("GET", orgEndpoint, &org, Config.DefaultTimeoutDuration())
	Expect(org.Resources).ToNot(BeEmpty())
	quotaGUID := org.Resources[0].Relationships.Quota.Data.GUID
	Expect(quotaGUID).ToNot(BeEmpty())

	var quota struct {
		Name string `json:"name"`
	}
	ApiRequest("GET", fmt.Sprintf("/v3/organization_quotas/%s", quotaGUID), &quota, Config.DefaultTimeoutDuration())
	Expect(quota.Name).ToNot(BeEmpty())

	AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("update-quota", quota.Name,
			"-m", memoryLimit).Wait()).To(Exit(0))
	})
}
