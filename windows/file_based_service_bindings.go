package windows

import (
	"fmt"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FileBasedServiceBindingsDescribe("File Based Service Bindings", WindowsLifecycle, func() {
	var appName, serviceName, serviceGuid, appGuid, appFeatureFlag string

	JustBeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		serviceName = generator.PrefixedRandomName("cats", "svin")

		tags := "list, of, tags"
		creds := `{"username": "admin", "password":"pa55woRD"}`
		Expect(cf.Cf("create-user-provided-service", serviceName, "-p", creds, "-t", tags).Wait()).To(Exit(0))
		serviceGuid = services.GetServiceInstanceGuid(serviceName)

		Expect(cf.Cf("create-app", appName).Wait()).To(Exit(0))
		appGuid = app_helpers.GetAppGuid(appName)

		appFeatureUrl := fmt.Sprintf("/v3/apps/%s/features/%s", appGuid, appFeatureFlag)
		Expect(cf.Cf("curl", appFeatureUrl, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()).To(Exit(0))

		Expect(cf.Cf("bind-service", appName, serviceName).Wait()).To(Exit(0))

		Expect(cf.Cf(app_helpers.WindowsCatnipWithArgs(
			appName,
			"-m", DEFAULT_MEMORY_LIMIT)...,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	})

	Context("When the file-based-vcap-services feature enabled", func() {

		BeforeEach(func() {
			appFeatureFlag = "file-based-vcap-services"
		})

		It("It should store the VCAP_SERVICE binding information in file in the VCAP_SERVICES_FILE_PATH", func() {
			services.ValidateFileBasedVcapServices(appName, serviceName, appGuid, serviceGuid)

		})
	},
	)

	Context("When the service-binding-k8s feature enabled", func() {

		BeforeEach(func() {
			appFeatureFlag = "service-binding-k8s"
		})
		It("It should have environment variable SERVICE_BINDING_ROOT which defines the location for the service binding", func() {
			services.ValidateServiceBindingK8s(appName, serviceName, appGuid, serviceGuid)
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("unbind-service", appName, serviceName).Wait()).Should(Exit(0))
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete-service", serviceName, "-f").Wait()).Should(Exit(0))
	})

})
