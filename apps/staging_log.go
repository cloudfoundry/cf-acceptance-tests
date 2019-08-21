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
		app_helpers.AppReport(appName)

		cf.Cf("delete", appName, "-f", "-r").Wait()
	})

	It("has its staging log streamed during a push", func() {
		push := cf.Cf("push",
			appName,
			"-b", Config.GetBinaryBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Catnip,
			"-c", "./catnip",
			"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(0))

		output := string(push.Out.Contents())
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
