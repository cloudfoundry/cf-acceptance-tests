package capi_experimental

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	"github.com/mholt/archiver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	staticFileZip = "assets/staticfile.zip"
)

var _ = CapiExperimentalDescribe("deployment", func() {

	var (
		appName              string
		appGuid              string
		dropletGuid          string
		spaceGuid            string
		spaceName            string
		token                string
		instances            int
		webProcess           Process
		stopCheckingAppAlive chan<- bool
		appCheckerIsDone     <-chan bool
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")
		spaceName = TestSetup.RegularUserContext().Space
		spaceGuid = GetSpaceGuidFromName(spaceName)
		appGuid = CreateApp(appName, spaceGuid, `{"foo":"bar"}`)
		token = GetAuthToken()
		dropletGuid = uploadDroplet(appGuid, assets.NewAssets().DoraZip, Config.GetRubyBuildpackName(), token)
		AssignDropletToApp(appGuid, dropletGuid)

		processes := GetProcesses(appGuid, appName)
		webProcess = GetProcessByType(processes, "web")

		CreateAndMapRoute(appGuid, spaceName, Config.GetAppsDomain(), appName)
		instances = 2
		ScaleApp(appGuid, instances)

		StartApp(appGuid)
		Expect(string(cf.Cf("apps").Wait().Out.Contents())).To(MatchRegexp(fmt.Sprintf("(v3-)?(%s)*(-web)?(\\s)+(started)", webProcess.Name)))

		By("waiting until all instances are running")
		Eventually(func() int {
			guid := GetProcessGuidForType(appGuid, "web")
			Expect(guid).ToNot(BeEmpty())
			return GetRunningInstancesStats(guid)
		}).Should(Equal(instances))

		By("Creating a second droplet for the app")
		makeStaticFileZip()
		dropletGuid = uploadDroplet(appGuid, staticFileZip, Config.GetStaticFileBuildpackName(), token)
	})

	AfterEach(func() {
		FetchRecentLogs(appGuid, token, Config)
		DeleteApp(appGuid)
		os.Remove("assets/staticfile.zip")
	})

	Describe("Deployment", func() {
		BeforeEach(func() {
			By("Assigning a second droplet for the app")
			AssignDropletToApp(appGuid, dropletGuid)
			stopCheckingAppAlive, appCheckerIsDone = checkAppRemainsAlive(appName)
		})

		AfterEach(func() {
			stopCheckingAppAlive <- true
			<-appCheckerIsDone
		})

		It("deploys an app with no downtime", func() {
			Eventually(func() string {
				return helpers.CurlAppRoot(Config, appName)
			}).Should(ContainSubstring("Hi, I'm Dora"))

			_, originalWorkerStartEvent := GetLastAppUseEventForProcess("worker", "STARTED", "")

			deploymentGuid := CreateDeployment(appGuid)
			Expect(deploymentGuid).ToNot(BeEmpty())
			webishProcessType := fmt.Sprintf("web-deployment-%s", deploymentGuid)

			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, "web")
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(Equal(1))

			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, webishProcessType)
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(BeNumerically(">", 0))

			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, "web")
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(Equal(0))

			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, "web")
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(BeNumerically(">", 0))

			Eventually(func() string {
				return GetProcessGuidForType(appGuid, webishProcessType)
			}).Should(BeEmpty())

			counter := 0
			Eventually(func() int {
				if strings.Contains(helpers.CurlAppRoot(Config, appName), "Hello from a staticfile") {
					counter++
				} else {
					counter = 0
				}
				return counter
			}).Should(Equal(10))

			Eventually(func() bool {
				restartEventExists, _ := GetLastAppUseEventForProcess("worker", "STARTED", originalWorkerStartEvent.Metadata.Guid)
				return restartEventExists
			}).Should(BeTrue(), "Did not find a start event indicating the 'worker' process restarted")
		})
	})

	Describe("cancelling deployments", func() {
		It("rolls back to the previous droplet", func() {
			By("creating a deployment with the second droplet")
			deploymentGuid := CreateDeploymentForDroplet(appGuid, dropletGuid)
			Expect(deploymentGuid).ToNot(BeEmpty())
			webishProcessType := fmt.Sprintf("web-deployment-%s", deploymentGuid)

			By("waiting until there is a webish process before canceling")
			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, webishProcessType)
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(BeNumerically(">", 0))

			By("canceling the deployment")
			CancelDeployment(deploymentGuid)

			By("waiting until there are no webish processes")
			Eventually(func() string {
				return GetProcessGuidForType(appGuid, webishProcessType)
			}).Should(BeEmpty())

			Eventually(func() int {
				guid := GetProcessGuidForType(appGuid, "web")
				Expect(guid).ToNot(BeEmpty())
				return GetRunningInstancesStats(guid)
			}).Should(Equal(instances))

			counter := 0
			Eventually(func() int {
				if strings.Contains(helpers.CurlAppRoot(Config, appName), "Hi, I'm Dora") {
					counter++
				} else {
					counter = 0
				}
				return counter
			}).Should(Equal(10))
		})
	})
})

func checkAppRemainsAlive(appName string) (chan<- bool, <-chan bool) {
	doneChannel := make(chan bool, 1)
	appCheckerIsDone := make(chan bool, 1)
	ticker := time.NewTicker(1 * time.Second)
	tickerChannel := ticker.C

	go func() {
		defer GinkgoRecover()
		for {
			select {
			case <-doneChannel:
				ticker.Stop()
				appCheckerIsDone <- true
				return
			case <-tickerChannel:
				Expect(helpers.CurlAppRoot(Config, appName)).ToNot(ContainSubstring("404 Not Found"))
			}
		}
	}()

	return doneChannel, appCheckerIsDone
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

func uploadDroplet(appGuid, zipFile, buildpackName, token string) string {
	packageGuid := CreatePackage(appGuid)
	url := fmt.Sprintf("%s%s/v3/packages/%s/upload", Config.Protocol(), Config.GetApiEndpoint(), packageGuid)

	UploadPackage(url, zipFile, token)
	WaitForPackageToBeReady(packageGuid)

	buildGuid := StageBuildpackPackage(packageGuid, buildpackName)
	WaitForBuildToStage(buildGuid)
	return GetDropletFromBuild(buildGuid)
}
