package credhub

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
)

var _ = CredhubDescribe("service keys", func() {
	var (
		chBrokerAppName string
		chServiceName   string
		instanceName    string
		serviceKeyName  string
	)

	BeforeEach(func() {
		TestSetup.RegularUserContext().TargetSpace()
		cf.Cf("target", "-o", TestSetup.RegularUserContext().Org)
		Expect(string(cf.Cf("running-environment-variable-group").Wait(Config.DefaultTimeoutDuration()).Out.Contents())).To(ContainSubstring("CREDHUB_API"), "CredHub API environment not set")

		chBrokerAppName = random_name.CATSRandomName("BRKR-CH")

		Expect(cf.Cf(
			"push", chBrokerAppName,
			"-b", Config.GetGoBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().CredHubServiceBroker,
			"-f", assets.NewAssets().CredHubServiceBroker+"/manifest.yml",
			"-d", Config.GetAppsDomain(),
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed pushing credhub-enabled service broker")

		chServiceName = random_name.CATSRandomName("SERVICE-NAME")
		Expect(cf.Cf(
			"set-env", chBrokerAppName,
			"SERVICE_NAME", chServiceName,
		).Wait(Config.DefaultTimeoutDuration())).To(Exit(0), "failed setting SERVICE_NAME env var on credhub-enabled service broker")

		Expect(cf.Cf(
			"restart", chBrokerAppName,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0), "failed restarting credhub-enabled service broker")

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			serviceUrl := "https://" + chBrokerAppName + "." + Config.GetAppsDomain()
			createServiceBroker := cf.Cf("create-service-broker", chBrokerAppName, Config.GetAdminUser(), Config.GetAdminPassword(), serviceUrl).Wait(Config.DefaultTimeoutDuration())
			Expect(createServiceBroker).To(Exit(0), "failed creating credhub-enabled service broker")

			enableAccess := cf.Cf("enable-service-access", chServiceName, "-o", TestSetup.RegularUserContext().Org).Wait(Config.DefaultTimeoutDuration())
			Expect(enableAccess).To(Exit(0), "failed to enable service access for credhub-enabled broker")

			TestSetup.RegularUserContext().TargetSpace()
			instanceName = random_name.CATSRandomName("SVIN-CH")
			createService := cf.Cf("create-service", chServiceName, "credhub-read-plan", instanceName).Wait(Config.DefaultTimeoutDuration())
			Expect(createService).To(Exit(0), "failed creating credhub enabled service")
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(chBrokerAppName, Config.DefaultTimeoutDuration())

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			TestSetup.RegularUserContext().TargetSpace()

			Expect(cf.Cf("delete-service-key", instanceName, serviceKeyName, "-f").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("purge-service-instance", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("delete-service-broker", chBrokerAppName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})
	})

	Context("when a service key for a service instance is requested from a CredHub-enabled broker", func() {
		It("Cloud Controller retrieves the value from CredHub for the service key", func() {
			TestSetup.RegularUserContext().TargetSpace()

			serviceKeyName = random_name.CATSRandomName("SVKEY-CH")
			createKey := cf.Cf("create-service-key", instanceName, serviceKeyName).Wait(Config.DefaultTimeoutDuration())
			Expect(createKey).To(Exit(0), "failed to create key")

			keyInfo := cf.Cf("service-key", instanceName, serviceKeyName).Wait(Config.DefaultTimeoutDuration())
			Expect(keyInfo).To(Exit(0), "failed key info")

			Expect(keyInfo).To(Say(`"password": "rainbowDash"`))
			Expect(keyInfo).To(Say(`"user-name": "pinkyPie"`))
		})
	})
})
