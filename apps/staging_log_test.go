package apps

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("An application being staged", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)
	})

	It("has its staging log streamed during a push", func() {
		push := cf.Cf("push", appName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)

		output := string(push.Buffer().Contents())
		expected := []string{"Installing dependencies", "Uploading droplet", "App started"}
		found := false
		for _, value := range expected {
			if strings.Contains(output, value) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "Did not find one of the expected log lines: %s", expected)
	})

	Context("when staging fails, error logs should be streamed", func() {

		It("fails when memory limit required by app exceeds org memory", func() {
			push := cf.Cf("push", appName, "-p", assets.NewAssets().Dora, "-m", helpers.RUNAWAY_QUOTA_MEM_LIMIT).Wait(CF_PUSH_TIMEOUT)

			output := string(push.Buffer().Contents())
			Eventually(push).Should(Exit(1))
			Expect(output).To(ContainSubstring("FAILED"))
			Expect(output).To(ContainSubstring("100005"))
			Expect(output).To(ContainSubstring("You have exceeded your organization's memory limit"))

			app := cf.Cf("app", appName)
			Eventually(app).Should(Exit(0))
			Eventually(app.Out).Should(gbytes.Say("requested state: stopped"))
		})

	})
})
