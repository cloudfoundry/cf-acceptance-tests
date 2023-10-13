package apps

import (
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Rolling deploys", func() {
	var (
		appName string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Push(appName,
			"-p", assets.NewAssets().Dora,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(Config, appName)
		}).Should(ContainSubstring("Hi, I'm Dora!"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("deploys the app with zero downtime", func() {
		By("checking the app remains available")
		doneChannel := make(chan bool, 1)
		ticker := time.NewTicker(1 * time.Second)
		tickerChannel := ticker.C

		defer func() {
			ticker.Stop()
			close(doneChannel)
		}()

		go func() {
			defer GinkgoRecover()

			for {
				select {
				case <-doneChannel:
					ticker.Stop()
					return
				case <-tickerChannel:
					appResponse := helpers.CurlAppRoot(Config, appName)
					Expect(appResponse).ToNot(ContainSubstring("404"))
					Expect(appResponse).To(ContainSubstring("Hi, I'm Dora!"))
				}
			}
		}()

		Expect(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Dora,
			"--strategy=rolling",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})
})
