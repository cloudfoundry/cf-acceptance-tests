package apps

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Large_payload", func() {
	var appName string
	AfterEach(func() {
		app_helpers.AppReport(appName, DEFAULT_TIMEOUT)

		Expect(cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})
	It("should be able to curl for a large response body", func() {
		appName := generator.PrefixedRandomName("CATS-APPS-")
		Expect(cf.Cf("push", appName, "--no-start", "-b", config.RubyBuildpackName, "-m", "128M", "-p", assets.NewAssets().Dora, "-d", config.AppsDomain).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		app_helpers.ConditionallyEnableDiego(appName)
		Expect(cf.Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() int {
			curlResponse := helpers.CurlApp(appName, fmt.Sprintf("/largetext/5"))
			return len(curlResponse)
		}, 10*time.Second, 10*time.Second).Should(Equal(5 * 1024))
	})
})
