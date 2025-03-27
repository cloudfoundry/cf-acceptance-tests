package file_based_service_bindings

import (
	"fmt"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type LifeCycle interface {
	Prepare(string, string, string) (string, string)
}

type BuildpackLifecycles struct{}
type CNBLifecycles struct{}
type DockerLifecycles struct{}

func (b *BuildpackLifecycles) Prepare(serviceName, appName, appFeatureFlag string) (string, string) {

	Expect(cf.Cf("create-app", appName).Wait()).To(Exit(0))
	appGuid, serviceGuid := LifeCycleCommon(serviceName, appName, appFeatureFlag)

	Expect(cf.Cf(app_helpers.CatnipWithArgs(
		appName,
		"-m", DEFAULT_MEMORY_LIMIT)...,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	return appGuid, serviceGuid

}

func (c *CNBLifecycles) Prepare(serviceName, appName, appFeatureFlag string) (string, string) {

	Expect(cf.Cf("create-app", appName, "--app-type", "cnb", "--buildpack", Config.GetGoBuildpackName()).Wait()).To(Exit(0))
	appGuid, serviceGuid := LifeCycleCommon(serviceName, appName, appFeatureFlag)

	Expect(cf.Cf(
		"push",
		appName,
		"--lifecycle", "cnb",
		"--buildpack", Config.GetCNBGoBuildpackName(),
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", assets.NewAssets().CatnipSrc,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	return appGuid, serviceGuid
}

func (d *DockerLifecycles) Prepare(serviceName, appName, appFeatureFlag string) (string, string) {

	Expect(cf.Cf("create-app", appName, "--app-type", "docker").Wait()).To(Exit(0))
	appGuid, serviceGuid := LifeCycleCommon(serviceName, appName, appFeatureFlag)

	Expect(cf.Cf(
		"push",
		appName,
		"--docker-image", Config.GetCatnipDockerAppImage(),
		"-m", DEFAULT_MEMORY_LIMIT,
	).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	return appGuid, serviceGuid
}

const (
	TAGS  = "list, of, tags"
	CREDS = `{"username": "admin", "password":"pa55woRD"}`
)

func LifeCycleCommon(serviceName, appName, appFeatureFlag string) (string, string) {

	Expect(cf.Cf("create-user-provided-service", serviceName, "-p", CREDS, "-t", TAGS).Wait()).To(Exit(0))
	serviceGuid := services.GetServiceInstanceGuid(serviceName)

	appGuid := app_helpers.GetAppGuid(appName)

	appFeatureUrl := fmt.Sprintf("/v3/apps/%s/features/%s", appGuid, appFeatureFlag)
	Expect(cf.Cf("curl", appFeatureUrl, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()).To(Exit(0))

	Expect(cf.Cf("bind-service", appName, serviceName).Wait()).To(Exit(0))

	return appGuid, serviceGuid
}
