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
	"os"
	"path/filepath"
)

type LifeCycle interface {
	CreateAppArgs() []string
	PushArgs() []string
}

type BuildpackLifecycles struct{}
type CNBLifecycles struct{}
type DockerLifecycles struct{}

func (b *BuildpackLifecycles) CreateAppArgs() []string {
	return []string{}
}

func (b *BuildpackLifecycles) PushArgs() []string {
	return []string{
		"--buildpack", Config.GetBinaryBuildpackName(),
		"-p", assets.NewAssets().Catnip,
		"-c", "./catnip",
	}
}

func (c *CNBLifecycles) CreateAppArgs() []string {
	return []string{
		"--app-type", "cnb",
		"--buildpack", Config.GetCNBGoBuildpackName(),
	}
}

func (c *CNBLifecycles) PushArgs() []string {
	return []string{
		"--lifecycle", "cnb",
		"--buildpack", Config.GetCNBGoBuildpackName(),
		"-p", assets.NewAssets().CatnipSrc,
	}
}

func (d *DockerLifecycles) CreateAppArgs() []string {
	return []string{
		"--app-type", "docker",
	}
}

func (d *DockerLifecycles) PushArgs() []string {
	return []string{
		"--docker-image", Config.GetCatnipDockerAppImage(),
	}
}

const (
	TAGS  = "list, of, tags"
	CREDS = `{"username": "admin", "password":"pa55woRD"}`
)

func Prepare(appName, serviceName, appFeatureFlag string, lifecycle LifeCycle) (string, string) {
	appGuid := CreateApp(appName, lifecycle.CreateAppArgs()...)
	serviceGuid := CreateUpsi(serviceName)
	BindUpsi(appName, serviceName)
	EnableFeatureViaAPI(appGuid, appFeatureFlag)

	pushArgs := append([]string{appName}, lifecycle.PushArgs()...)
	Push(pushArgs...)

	return appGuid, serviceGuid
}

func PrepareWithManifest(appName, serviceName, appFeatureFlag string, lifecycle LifeCycle) (string, string) {
	serviceGuid := CreateUpsi(serviceName)
	manifestFile := CreateManifest(appName, serviceName, appFeatureFlag)

	pushArgs := append([]string{"-f", manifestFile}, lifecycle.PushArgs()...)
	Push(pushArgs...)

	return app_helpers.GetAppGuid(appName), serviceGuid
}

func CreateApp(appName string, args ...string) string {
	createAppArgs := []string{
		"create-app", appName,
	}
	createAppArgs = append(createAppArgs, args...)
	Expect(cf.Cf(createAppArgs...).Wait()).To(Exit(0))
	appGuid := app_helpers.GetAppGuid(appName)

	return appGuid
}

func CreateUpsi(serviceName string) string {
	Expect(cf.Cf("create-user-provided-service", serviceName, "-p", CREDS, "-t", TAGS).Wait()).To(Exit(0))
	serviceGuid := services.GetServiceInstanceGuid(serviceName)

	return serviceGuid
}

func CreateManifest(appName, serviceName, appFeatureFlag string) string {
	tmpdir, err := os.MkdirTemp(os.TempDir(), appName)
	Expect(err).ToNot(HaveOccurred())

	manifestFile := filepath.Join(tmpdir, "manifest.yml")
	manifestContent := fmt.Sprintf(`---
applications:
- name: %s
  features:
    %s: true
  services:
    - %s
`, appName, appFeatureFlag, serviceName)
	err = os.WriteFile(manifestFile, []byte(manifestContent), 0644)
	Expect(err).ToNot(HaveOccurred())

	return manifestFile
}

func BindUpsi(appName, serviceName string) {
	Expect(cf.Cf("bind-service", appName, serviceName).Wait()).To(Exit(0))
}

func EnableFeatureViaAPI(appGuid, appFeatureFlag string) {
	appFeatureUrl := fmt.Sprintf("/v3/apps/%s/features/%s", appGuid, appFeatureFlag)
	Expect(cf.Cf("curl", appFeatureUrl, "-X", "PATCH", "-d", `{"enabled": true}`).Wait()).To(Exit(0))
}

func Push(args ...string) {
	pushArgs := []string{
		"push",
		"-m", DEFAULT_MEMORY_LIMIT,
	}
	pushArgs = append(pushArgs, args...)
	Expect(cf.Cf(pushArgs...).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
}
