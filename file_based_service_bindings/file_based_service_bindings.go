package file_based_service_bindings

import (
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/generator"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a buildpack app", BuildpackLifecycle, func() {
	callback(&BuildpackLifecycles{})
})

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a CNB app", CNBLifecycle, func() {
	callback(&CNBLifecycles{})
})

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a Docker app", DockerLifecycle, func() {
	callback(&DockerLifecycles{})
})

var callback = func(lifeCycle LifeCycle) {
	var appName, serviceName string
	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		serviceName = generator.PrefixedRandomName("cats", "svin") // uppercase characters are not valid
	})

	Context("When the file-based-vcap-services feature enabled", func() {
		It("It should store the VCAP_SERVICE binding information in file in the VCAP_SERVICES_FILE_PATH", func() {
			appGuid, serviceGuid := lifeCycle.Prepare(serviceName, appName, "file-based-vcap-services")
			services.ValidateFileBasedVcapServices(appName, serviceName, appGuid, serviceGuid)

		})
	})

	Context("When the service-binding-k8s feature enabled", func() {
		It("It should have environment variable SERVICE_BINDING_ROOT which defines the location for the service binding", func() {
			appGuid, serviceGuid := lifeCycle.Prepare(serviceName, appName, "service-binding-k8s")
			services.ValidateServiceBindingK8s(appName, serviceName, appGuid, serviceGuid)
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("unbind-service", appName, serviceName).Wait()).Should(Exit(0))
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete-service", serviceName, "-f").Wait()).Should(Exit(0))
	})

}
