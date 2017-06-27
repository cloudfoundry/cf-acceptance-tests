package nimbus

import (

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)


var _ = NimbusDescribe("rabbitmq service federation", func() {

	var appName, redisName, rabbitName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		redisName = random_name.CATSRandomName("SVC")
		rabbitName = random_name.CATSRandomName("SVC")

		Expect(cf.Cf("create-service", "cf-redis", "shared-vm", redisName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("create-service", "l2-rabbitmq", "default", rabbitName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().NimbusServices, "--no-start", "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("bind-service", appName, redisName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("bind-service", appName, rabbitName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", redisName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", rabbitName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})


	XIt("app instances in both data centres receive messages", func() {

		Eventually(func() string {
			helpers.CurlApp(Config, appName, "/rabbit/publish")
			return helpers.CurlApp(Config, appName, "/rabbit/check/2")
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("OK"))
	})

})
