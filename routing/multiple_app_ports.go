package routing

import (
	"fmt"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = RoutingDescribe("Multiple App Ports", func() {
	SkipOnK8s("Not yet supported in CF-for-K8s")

	var (
		appName             string
		secondRouteHostname string
		multiPortAppAsset   = assets.NewAssets().MultiPortApp
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		cmd := fmt.Sprintf("go-online --ports=7777,8888,8080")

		Expect(cf.Cf("push",
			appName,
			"-b", Config.GetGoBuildpackName(),
			"-c", cmd,
			"-f", filepath.Join(multiPortAppAsset, "manifest.yml"),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", multiPortAppAsset,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	Context("when app only has single route", func() {
		Context("when no ports are specified for the app", func() {
			It("should listen on the default app port", func() {
				Eventually(func() string {
					return helpers.CurlApp(Config, appName, "/port")
				}).Should(ContainSubstring("8080"))
			})
		})
	})

	Context("when app has multiple ports mapped", func() {
		BeforeEach(func() {
			appGUID := app_helpers.GetAppGuid(appName)

			secondRouteHostname = fmt.Sprintf("%s-two", appName)
			Expect(cf.Cf("create-route", Config.GetAppsDomain(),
				"--hostname", secondRouteHostname,
			).Wait()).To(Exit(0))

			destination := Destination{
				App: App{
					GUID: appGUID,
				},
				Port: 7777,
			}
			InsertDestinations(GetRouteGuid(secondRouteHostname), []Destination{destination})

			Expect(cf.Cf("restart", appName, "--strategy", "rolling").Wait()).To(Exit(0))
		})

		It("should listen on multiple ports", func() {
			Eventually(func() string {
				return helpers.CurlApp(Config, secondRouteHostname, "/port")
			}).Should(ContainSubstring("7777"))

			Consistently(func() string {
				return helpers.CurlApp(Config, appName, "/port")
			}, Config.SleepTimeoutDuration(), "5s").Should(ContainSubstring("8080"))
		})
	})
})
