package capi_experimental


import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"github.com/mholt/archiver"
	"os"
	"time"
)

var (
	staticFileZip = "assets/staticfile.zip"
)

var _ = CapiExperimentalDescribe("deployment", func() {

	var (
		appName     string
		appGuid     string
		spaceGuid   string
		spaceName   string
		token       string
		webProcess  Process
		stopCheckingAppAlive chan<- bool
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGuid = GetSpaceGuidFromName(spaceName)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		token = GetAuthToken()
		uploadAndAssignDroplet(appGuid, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName(), token)

		processes := GetProcesses(appGuid, appName)
		webProcess = GetProcessByType(processes, "web")

		CreateAndMapRoute(appGuid, spaceName, Config.GetAppsDomain(), appName)
		ScaleApp(appGuid, 2)

		StartApp(appGuid)
		Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

		By("Creating and assigning a second droplet for the app")
		makeStaticFileZip()
		uploadAndAssignDroplet(appGuid, staticFileZip, Config.GetStaticFileBuildpackName(), token)

		stopCheckingAppAlive = checkAppRemainsAlive(appName)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		stopCheckingAppAlive <- true
		DeleteApp(appGuid)
		os.Remove("assets/staticfile.zip")
	})

	Describe("Deployment", func() {
		It("deploys an app with no downtime", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora!"))

			deploymentGuid := CreateDeployment(appGuid)
			webishProcessType := fmt.Sprintf("web-deployment-%s", deploymentGuid)

			Eventually(func() int {
				return GetRunningInstancesStats(GetProcessGuidForType(appGuid, "web"))
			}).Should(Equal(1))

			Eventually(func() int {
				return GetRunningInstancesStats(GetProcessGuidForType(appGuid, webishProcessType))
			}).Should(BeNumerically(">", 0))

			Eventually(func() int {
				return GetRunningInstancesStats(GetProcessGuidForType(appGuid, "web"))
			}).Should(Equal(0))

			Eventually(func() int {
				return GetRunningInstancesStats(GetProcessGuidForType(appGuid, webishProcessType))
			}).Should(Equal(2))

			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hello from a staticfile"))
		})
	})
})



func checkAppRemainsAlive(appName string) chan<- bool {
	doneChannel := make(chan bool, 1)
	ticker := time.NewTicker(1 * time.Second)
	tickerChannel := ticker.C

	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-doneChannel:
				ticker.Stop()
				return
			case <-tickerChannel:
				Expect(helpers.CurlAppRoot(Config, appName)).ToNot(ContainSubstring("404 Not Found"))
			}
		}
	}()

	return doneChannel
}


func makeStaticFileZip() {
	staticFiles, err := ioutil.ReadDir(assets.NewAssets().Staticfile)
	Expect(err).NotTo(HaveOccurred())

	var staticFileNames []string
	for _, staticFile := range staticFiles {
		staticFileNames = append(staticFileNames, assets.NewAssets().Staticfile+"/"+staticFile.Name())
	}

	err = archiver.Zip.Make(staticFileZip, staticFileNames)
	Expect(err).NotTo(HaveOccurred())
}

func uploadAndAssignDroplet(appGuid, zipFile, buildpackName, token string) {
	packageGuid := CreatePackage(appGuid)
	url := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

	UploadPackage(url, zipFile, token)
	WaitForPackageToBeReady(packageGuid)

	buildGuid := StageBuildpackPackage(packageGuid, buildpackName)
	WaitForBuildToStage(buildGuid)
	dropletGuid := GetDropletFromBuild(buildGuid)
	AssignDropletToApp(appGuid, dropletGuid)
}
