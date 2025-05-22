package file_based_service_bindings

import (
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

	Context("When the file-based-vcap-services feature is enabled", func() {
		Context("Via API call", func() {
			It("It should store the VCAP_SERVICES binding information in a file in the VCAP_SERVICES_FILE_PATH", func() {
				appGuid, serviceGuid := Prepare(appName, serviceName, "file-based-vcap-services", lifeCycle)
				services.ValidateFileBasedVcapServices(appName, serviceName, appGuid, serviceGuid)
			})
		})

		Context("Via manifest", func() {
			It("It should store the VCAP_SERVICES binding information in a file in the VCAP_SERVICES_FILE_PATH", func() {
				appGuid, serviceGuid := PrepareWithManifest(appName, serviceName, "file-based-vcap-services", lifeCycle)
				services.ValidateFileBasedVcapServices(appName, serviceName, appGuid, serviceGuid)
			})
		})
	})

	Context("When the service-binding-k8s feature is enabled", func() {
		Context("Via API call", func() {
			It("It should store the binding information in files under the SERVICE_BINDING_ROOT path", func() {
				appGuid, serviceGuid := Prepare(appName, serviceName, "service-binding-k8s", lifeCycle)
				services.ValidateServiceBindingK8s(appName, serviceName, appGuid, serviceGuid)
			})
		})

		Context("Via manifest", func() {
			It("It should store the binding information in files under the SERVICE_BINDING_ROOT path", func() {
				appGuid, serviceGuid := PrepareWithManifest(appName, serviceName, "service-binding-k8s", lifeCycle)
				services.ValidateServiceBindingK8s(appName, serviceName, appGuid, serviceGuid)
			})
		})
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("unbind-service", appName, serviceName).Wait()).Should(Exit(0))
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete-service", serviceName, "-f").Wait()).Should(Exit(0))
	})
}
