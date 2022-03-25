package apps

import (
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/cf"
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
		push := cf.Cf(app_helpers.CatnipWithArgs(appName, "-m", DEFAULT_MEMORY_LIMIT)...).Wait(Config.CfPushTimeoutDuration())
		Expect(push).To(Exit(0))

		output := string(push.Out.Contents())
		var expected []string
		if !Config.RunningOnK8s() {
			expected = []string{"Installing dependencies", "Uploading droplet", "App started"}
		} else {
			expected = []string{"Paketo Procfile Buildpack", "Build successful"}
		}
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
