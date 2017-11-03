package nimbus

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"time"
)

var _ = NimbusDescribe("nb-config", func() {

	var appName string

	BeforeEach(func() {

		if Config.GetIncludeNimbusNBConfig() != true {
			Skip("include_nimbus_nb_config was not set to true")
		}

		appName = random_name.CATSRandomName("APP")
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().NimbusServices, "--no-start", "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("opens container firewalls to allow backend calls to itself", func() {

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/nbconfig/test")
		}, 10*time.Second).Should(ContainSubstring("OK"))
	})

})
