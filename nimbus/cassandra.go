package nimbus

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = NimbusDescribe("cassandra service", func() {

	var appName, cassandraName string

	BeforeEach(func() {

		if Config.GetIncludeNimbusServiceCassandra() != true {
			Skip("include_nimbus_service_cassandra was not set to true")
		}

		appName = random_name.CATSRandomName("APP")
		cassandraName = random_name.CATSRandomName("SVC")

		Expect(cf.Cf("create-service", Config.GetNimbusServiceNameCassandra(), Config.GetNimbusServicePlanCassandra(), cassandraName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().NimbusServices, "--no-start", "-i", "2").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("bind-service", appName, cassandraName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))

		// mapping "app_name-[index]" route to be able to hit individual instances
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		Expect(cf.Cf("delete-service", cassandraName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	It("is accessible in hemel and slough datacenters", func() {

		randomKey := random_name.CATSRandomName("VAL")
		randomValue := random_name.CATSRandomName("KEY")

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/cassandra/insert/"+randomKey+"/"+randomValue)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("OK"))

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/cassandra/read/"+randomKey+"/"+randomValue)
		}, Config.DefaultTimeoutDuration()).Should(ContainSubstring("OK"))

	})

})
