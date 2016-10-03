package apps

import (
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = AppsDescribe("An application being staged", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())
	})

	It("has its staging log streamed during a push", func() {
		Eventually(cf.Cf("push", appName, "--no-start", "-b", Config.GetRubyBuildpackName(), "-m", DEFAULT_MEMORY_LIMIT, "-p", assets.NewAssets().Dora, "-d", Config.GetAppsDomain()), Config.DefaultTimeoutDuration()).Should(Exit(0))
		app_helpers.SetBackend(appName)
		start := cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())

		output := string(start.Buffer().Contents())
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
})
