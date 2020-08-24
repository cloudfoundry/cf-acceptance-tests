package apps

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = AppsDescribe("Piecemeal App Creation", func() {
	var (
		appName string
	)

	// Droplet tests aren't applicable to cf-for-k8s
	SkipOnK8s()

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	It("successfully creates and starts a new app with discrete commands", func() {
		Expect(cf.Cf("create-app", appName).Wait()).To(Exit(0))
		session := cf.Cf("app", appName, "--guid")
		Eventually(session).Should(Exit(0))
		appGUID := strings.TrimSpace(string(session.Buffer().Contents()))

		Expect(cf.Cf("create-package", appName, "-p", assets.NewAssets().Dora).Wait()).To(Exit(0))

		path := fmt.Sprintf("v3/apps/%s/packages", appGUID)
		listPackages := cf.Cf("curl", path).Wait().Out.Contents()
		var packageJSON struct {
			Resources []struct {
				PackageGUID string `json:"guid"`
			} `json:"resources"`
		}
		Expect(json.Unmarshal([]byte(listPackages), &packageJSON)).To(Succeed())

		Expect(cf.Cf("set-env", appName, "FOO", "BAR").Wait()).To(Exit(0))

		Expect(cf.Cf("stage", appName, "--package-guid", packageJSON.Resources[0].PackageGUID).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

		path = fmt.Sprintf("v3/apps/%s/droplets", appGUID)
		dropletBody := cf.Cf("curl", path).Wait().Out.Contents()
		var dropletJSON struct {
			Resources []struct {
				DropletGUID string `json:"guid"`
			} `json:"resources"`
		}
		Expect(json.Unmarshal([]byte(dropletBody), &dropletJSON)).To(Succeed())

		Expect(cf.Cf("set-droplet", appName, dropletJSON.Resources[0].DropletGUID).Wait()).To(Exit(0))

		Expect(cf.Cf("start", appName).Wait()).To(Exit(0))

		Expect(cf.Cf("map-route", appName, Config.GetAppsDomain(), "--hostname", appName).Wait()).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlApp(Config, appName, "/env/FOO")
		}).Should(ContainSubstring("BAR"))
	})
})
