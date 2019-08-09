package windows

import (
	"fmt"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	. "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

const EXCEED_CELL_MEMORY = "900g"

var _ = WindowsDescribe("Getting instance information", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Push(appName,
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
			"-p", assets.NewAssets().WindowsWebapp,
			"-c", ".\\webapp.exe",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
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
	type quotaDefinitionUrl struct {
		Resources []struct {
			Entity struct {
				QuotaDefinitionUrl string `json:"quota_definition_url"`
			} `json:"entity"`
		} `json:"resources"`
	}
	orgEndpoint := fmt.Sprintf("/v2/organizations?q=name%%3A%s", TestSetup.GetOrganizationName())
	var org quotaDefinitionUrl
	ApiRequest("GET", orgEndpoint, &org, Config.DefaultTimeoutDuration())
	Expect(org.Resources).ToNot(BeEmpty())

	type quotaDefinition struct {
		Entity struct {
			Name string `json:"name"`
		} `json:"entity"`
	}
	var quota quotaDefinition
	ApiRequest("GET", org.Resources[0].Entity.QuotaDefinitionUrl, &quota, Config.DefaultTimeoutDuration())
	Expect(quota.Entity.Name).ToNot(BeEmpty())

	AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("update-quota", quota.Entity.Name,
			"-m", memoryLimit).Wait()).To(Exit(0))
	})
}
