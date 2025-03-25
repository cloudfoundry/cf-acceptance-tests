package file_based_service_bindings

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/generator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a buildpack app", BuildpackLifecycle, func() {
	callback(BuildpackLifecycle)
})

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a CNB app", CNBLifecycle, func() {
	callback(CNBLifecycle)
})

var _ = FileBasedServiceBindingsDescribe("Enabling file based service binding for a Docker app", DockerLifecycle, func() {
	callback(DockerLifecycle)
})

var callback = func(lifecycle string) {
	var appName, serviceName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		serviceName = generator.PrefixedRandomName("cats", "svin") // uppercase characters are not valid
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Eventually(cf.Cf("unbind-service", appName, serviceName).Wait()).Should(Exit(0))
		Eventually(cf.Cf("delete", appName, "-f")).Should(Exit(0))
		Eventually(cf.Cf("delete-service", serviceName, "-f").Wait()).Should(Exit(0))
	})

	It("creates the required files in the app container", func() {
		tags := "list, of, tags"
		creds := `{"username": "admin", "password":"pa55woRD"}`
		Expect(cf.Cf("create-user-provided-service", serviceName, "-p", creds, "-t", tags).Wait()).To(Exit(0))
		serviceGuid := services.GetServiceInstanceGuid(serviceName)

		if lifecycle == BuildpackLifecycle {
			Expect(cf.Cf("create-app", appName).Wait()).To(Exit(0))
		}
		if lifecycle == CNBLifecycle {
			Expect(cf.Cf("create-app", appName, "--app-type", "cnb", "--buildpack", Config.GetGoBuildpackName()).Wait()).To(Exit(0))
		}
		if lifecycle == DockerLifecycle {
			Expect(cf.Cf("create-app", appName, "--app-type", "docker").Wait()).To(Exit(0))
		}
		appGuid := app_helpers.GetAppGuid(appName)

		appFeatureUrl := fmt.Sprintf("/v3/apps/%s/features/file-based-service-bindings", appGuid)
		Expect(cf.Cf("curl", appFeatureUrl, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()).To(Exit(0))

		Expect(cf.Cf("bind-service", appName, serviceName).Wait()).To(Exit(0))

		if lifecycle == BuildpackLifecycle {
			Expect(cf.Cf(app_helpers.CatnipWithArgs(
				appName,
				"-m", DEFAULT_MEMORY_LIMIT)...,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		}
		if lifecycle == CNBLifecycle {
			Expect(cf.Cf(
				"push",
				appName,
				"--lifecycle", "cnb",
				"--buildpack", Config.GetCNBGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-p", assets.NewAssets().CatnipSrc,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		}
		if lifecycle == DockerLifecycle {
			Expect(cf.Cf(
				"push",
				appName,
				"--docker-image", Config.GetCatnipDockerAppImage(),
				"-m", DEFAULT_MEMORY_LIMIT,
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		}

		services.ValidateFileBasedServicebinding(appName, serviceName, appGuid, serviceGuid)
	})
}
